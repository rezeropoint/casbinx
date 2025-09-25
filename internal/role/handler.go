package role

import (
	"fmt"

	"casbinx/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// roleManager 角色权限管理器实现，处理角色特有的业务逻辑
type roleManager struct {
	enforcer          *core.Enforcer
	dbConn            sqlx.SqlConn
	securityValidator *core.SecurityValidator
}

// newRoleManager 创建角色权限管理器实现
func newRoleManager(dsn string, enforcer *core.Enforcer, securityValidator *core.SecurityValidator) (*roleManager, error) {
	// 初始化 PostgreSQL - 使用 URL 格式的 DSN
	dbConn := sqlx.NewSqlConn("postgres", dsn)

	// 创建管理器实例
	manager := &roleManager{
		enforcer:          enforcer,
		dbConn:            dbConn,
		securityValidator: securityValidator,
	}

	// 启动时初始化数据库表，如果失败则返回错误，让调用者决定如何处理
	if err := initDB(dbConn); err != nil {
		return nil, fmt.Errorf("角色管理器初始化失败，数据库表创建失败: %v", err)
	}

	return manager, nil
}

// CreateRole 创建自定义角色
func (m *roleManager) CreateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error {
	// 验证参数
	if roleKey == "" || roleName == "" {
		return core.ErrInvalidParameter
	}

	// 如果tenantKey为空，默认为全局角色
	if tenantKey == "" {
		return fmt.Errorf("租户键不能为空")
	}

	// 检查角色是否已存在
	exists, err := m.isRoleExists(roleKey)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("角色 '%s' 已存在", roleKey)
	}

	// 安全检查已在engine层处理

	// 创建角色元数据
	err = m.createRoleMetadata(roleKey, roleName, description, tenantKey, operatorKey)
	if err != nil {
		return fmt.Errorf("创建角色元数据失败: %v", err)
	}

	// 如果没有权限，添加一个占位权限来标识角色存在
	if len(permissions) == 0 {
		err = m.enforcer.AddPolicy(roleKey, tenantKey, core.Permission{Resource: core.ResourcePlaceholder, Action: core.ActionNone})
		if err != nil {
			// 回滚角色元数据
			m.deleteRoleMetadata(roleKey)
			return err
		}
		return nil
	}

	// 添加角色权限
	err = m.setRolePermissionsInTenant(roleKey, tenantKey, permissions)
	if err != nil {
		// 回滚角色元数据
		m.deleteRoleMetadata(roleKey)
		return err
	}

	return nil
}

// UpdateRole 更新自定义角色
func (m *roleManager) UpdateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error {
	// 验证参数
	if roleKey == "" {
		return core.ErrInvalidParameter
	}

	// tenantKey 不会为空（数据库约束 NOT NULL DEFAULT '*'）
	// 删除不必要的空值检查

	// 检查角色是否存在
	exists, err := m.isRoleExists(roleKey)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("角色 '%s' 不存在", roleKey)
	}

	// 检查是否为系统角色（包含系统权限）
	if hasSystemPerms, _ := m.HasSystemPermissions(roleKey); hasSystemPerms {
		return core.ErrSystemRoleImmutable
	}

	// 安全检查已在engine层处理

	// 更新角色元数据
	if roleName != "" || description != "" {
		err = m.updateRoleMetadata(roleKey, roleName, description)
		if err != nil {
			return fmt.Errorf("更新角色元数据失败: %v", err)
		}
	}

	// 更新角色权限
	return m.setRolePermissionsInTenant(roleKey, tenantKey, permissions)
}

// DeleteRole 删除自定义角色
func (m *roleManager) DeleteRole(roleKey string) error {
	// 验证参数
	if roleKey == "" {
		return core.ErrInvalidParameter
	}

	// 检查角色是否存在
	exists, err := m.isRoleExists(roleKey)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("角色 '%s' 不存在", roleKey)
	}

	// 检查是否为系统角色（包含系统权限）
	if hasSystemPerms, _ := m.HasSystemPermissions(roleKey); hasSystemPerms {
		return core.ErrSystemRoleImmutable
	}

	// 删除角色权限
	if err := m.enforcer.ClearPolicies(roleKey); err != nil {
		return err
	}

	// 删除角色元数据
	if err := m.deleteRoleMetadata(roleKey); err != nil {
		return fmt.Errorf("删除角色元数据失败: %v", err)
	}

	// 删除用户角色分配
	return nil
}

// GetRole 获取角色详情
func (m *roleManager) GetRole(roleKey string) (*core.Role, error) {
	if roleKey == "" {
		return nil, core.ErrInvalidParameter
	}

	// 从数据库获取角色元数据
	roleMetadata, err := m.getRoleMetadata(roleKey)
	if err != nil {
		return nil, fmt.Errorf("角色 '%s' 不存在", roleKey)
	}

	// 获取角色权限
	permissions, err := m.GetRolePermissions(roleKey)
	if err != nil {
		return nil, err
	}

	description := ""
	if roleMetadata.Description.Valid {
		description = roleMetadata.Description.String
	}

	return &core.Role{
		Key:         roleMetadata.RoleKey,
		Name:        roleMetadata.Name,
		Description: description,
		Permissions: permissions,
		TenantKey:   roleMetadata.TenantKey,
	}, nil
}

// ListRoles 获取角色列表
func (m *roleManager) ListRoles(tenantKey string, filter *core.RoleFilter) ([]*core.Role, error) {
	// 从数据库获取角色列表
	roleMetadataList, err := m.listRoleMetadata(tenantKey)
	if err != nil {
		return nil, err
	}

	var roles []*core.Role
	for _, roleMetadata := range roleMetadataList {
		// 获取角色权限
		permissions, err := m.GetRolePermissions(roleMetadata.RoleKey)
		if err != nil {
			continue // 跳过获取权限失败的角色
		}

		description := ""
		if roleMetadata.Description.Valid {
			description = roleMetadata.Description.String
		}

		role := &core.Role{
			Key:         roleMetadata.RoleKey,
			Name:        roleMetadata.Name,
			Description: description,
			Permissions: permissions,
			TenantKey:   roleMetadata.TenantKey,
		}

		// 应用过滤条件
		if m.matchRoleFilter(role, filter) {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// GetRolePermissions 获取角色权限
func (m *roleManager) GetRolePermissions(roleKey string) ([]core.Permission, error) {
	if roleKey == "" {
		return nil, core.ErrInvalidParameter
	}

	// 验证 roleKey 确实是角色（存在于 roles 表中）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return nil, err
	}
	if !isRole {
		return nil, fmt.Errorf("'%s' 不是一个有效的角色", roleKey)
	}

	// 获取角色权限
	policies, err := m.enforcer.GetPolicies(roleKey, "")
	if err != nil {
		return nil, err
	}

	permissions := make([]core.Permission, 0, len(policies))
	for _, policy := range policies {
		// 跳过占位权限
		if policy.Resource == core.ResourcePlaceholder && policy.Action == core.ActionNone {
			continue
		}

		permissions = append(permissions, core.Permission{
			Resource: policy.Resource,
			Action:   policy.Action,
		})
	}

	return permissions, nil
}

// GrantPermission 为角色授予权限
func (m *roleManager) GrantPermission(operatorKey, roleKey string, permission core.Permission) error {
	// 验证参数
	if roleKey == "" || permission.Resource == "" || permission.Action == "" {
		return core.ErrInvalidParameter
	}

	// 验证 roleKey 确实是角色（存在于 roles 表中）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return err
	}
	if !isRole {
		return fmt.Errorf("'%s' 不是一个有效的角色", roleKey)
	}

	// 检查是否为系统角色（包含系统权限）
	if hasSystemPerms, _ := m.HasSystemPermissions(roleKey); hasSystemPerms {
		return core.ErrSystemRoleImmutable
	}

	// 安全检查已在engine层处理

	// 获取角色的归属租户
	role, err := m.GetRole(roleKey)
	if err != nil {
		return err
	}

	// 使用角色的租户键（不会为空，数据库约束保证）
	tenantKey := role.TenantKey

	// 为角色添加权限（使用角色归属的租户域）
	return m.enforcer.AddPolicy(roleKey, tenantKey, permission)
}

// RevokePermission 撤销角色权限
func (m *roleManager) RevokePermission(operatorKey, roleKey string, permission core.Permission) error {
	// 验证参数
	if roleKey == "" || permission.Resource == "" || permission.Action == "" {
		return core.ErrInvalidParameter
	}

	// 验证 roleKey 确实是角色（存在于 roles 表中）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return err
	}
	if !isRole {
		return fmt.Errorf("'%s' 不是一个有效的角色", roleKey)
	}

	// 检查是否为系统角色（包含系统权限）
	if hasSystemPerms, _ := m.HasSystemPermissions(roleKey); hasSystemPerms {
		return core.ErrSystemRoleImmutable
	}

	// 安全检查已在engine层处理

	// 获取角色的归属租户
	role, err := m.GetRole(roleKey)
	if err != nil {
		return err
	}

	// 使用角色的租户键（不会为空，数据库约束保证）
	tenantKey := role.TenantKey

	// 撤销角色权限
	return m.enforcer.RemovePolicy(roleKey, tenantKey, permission)
}

// SetRolePermissions 设置角色的所有权限（替换现有权限）
func (m *roleManager) SetRolePermissions(roleKey string, permissions []core.Permission) error {
	if roleKey == "" {
		return core.ErrInvalidParameter
	}

	// 验证 roleKey 确实是角色（存在于 roles 表中）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return err
	}
	if !isRole {
		return fmt.Errorf("'%s' 不是一个有效的角色", roleKey)
	}

	// 检查是否为系统角色（包含系统权限）
	if hasSystemPerms, _ := m.HasSystemPermissions(roleKey); hasSystemPerms {
		return core.ErrSystemRoleImmutable
	}

	return m.setRolePermissions(roleKey, permissions)
}

// GetUsersWithRole 获取拥有指定角色的用户
func (m *roleManager) GetUsersWithRole(roleKey, tenantKey string) ([]string, error) {
	if roleKey == "" {
		return nil, core.ErrInvalidParameter
	}

	return m.enforcer.GetUsersWithRole(roleKey, tenantKey)
}

// HasSystemPermissions 检查角色是否包含系统权限
func (m *roleManager) HasSystemPermissions(roleKey string) (bool, error) {
	if roleKey == "" {
		return false, core.ErrInvalidParameter
	}

	// 验证 roleKey 确实是角色（存在于 roles 表中）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return false, err
	}
	if !isRole {
		// 如果不是角色，返回 false 而不是错误，避免影响其他逻辑
		return false, nil
	}

	// 获取角色权限（此时 GetRolePermissions 已经包含了角色验证）
	permissions, err := m.GetRolePermissions(roleKey)
	if err != nil {
		return false, err
	}

	// 检查是否包含系统权限
	for _, perm := range permissions {
		if m.isSystemPermission(perm) {
			return true, nil
		}
	}

	return false, nil
}

// UserRoleHasSystemPermissions 检查用户的角色是否包含系统权限
func (m *roleManager) UserRoleHasSystemPermissions(userKey, roleKey, tenantKey string) (bool, error) {
	if userKey == "" || roleKey == "" {
		return false, core.ErrInvalidParameter
	}

	// 验证 roleKey 确实是角色（存在于 roles 表中）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return false, err
	}
	if !isRole {
		// 如果不是角色，返回 false
		return false, nil
	}

	// 验证用户确实拥有该角色
	hasRole, err := m.enforcer.IsRoleAssigned(userKey, roleKey, tenantKey)
	if err != nil {
		return false, err
	}
	if !hasRole {
		return false, nil
	}

	// 检查角色是否包含系统权限（此时 HasSystemPermissions 已经包含了角色验证）
	return m.HasSystemPermissions(roleKey)
}

// GetAllGroupingPolicies 获取指定租户的所有角色分配
func (m *roleManager) GetAllGroupingPolicies(tenantKey string) ([]core.GroupingPolicy, error) {
	// 获取所有角色分组策略
	allGroupings, err := m.enforcer.GetGroupingPolicies()
	if err != nil {
		return nil, err
	}

	// 过滤指定租户的角色分配
	var filteredGroupings []core.GroupingPolicy
	for _, grouping := range allGroupings {
		// 如果指定了租户，只返回该租户或全局(*)的角色分配
		if tenantKey == "" || grouping.TenantKey == tenantKey || grouping.TenantKey == "*" {
			filteredGroupings = append(filteredGroupings, grouping)
		}
	}

	return filteredGroupings, nil
}
