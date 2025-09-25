package role

import (
	"github.com/rezeropoint/casbinx/core"
)

// setRolePermissions 设置角色权限（内部方法）
func (m *roleManager) setRolePermissions(roleKey string, permissions []core.Permission) error {
	// 先清除现有权限
	if err := m.enforcer.ClearPolicies(roleKey); err != nil {
		return err
	}

	// 添加新权限
	for _, perm := range permissions {
		if !perm.IsValid() {
			continue
		}

		if err := m.enforcer.AddPolicy(roleKey, "*", perm); err != nil {
			return err
		}
	}

	// 如果最终没有任何权限，添加占位权限
	if len(permissions) == 0 {
		return m.enforcer.AddPolicy(roleKey, "*", core.Permission{Resource: core.ResourcePlaceholder, Action: core.ActionNone})
	}

	return nil
}

// setRolePermissionsInTenant 在指定租户中设置角色权限
func (m *roleManager) setRolePermissionsInTenant(roleKey, tenantKey string, permissions []core.Permission) error {
	// 先清除现有权限
	if err := m.enforcer.ClearPolicies(roleKey); err != nil {
		return err
	}

	// 添加新权限（使用指定租户域）
	for _, perm := range permissions {
		if !perm.IsValid() {
			continue
		}

		if err := m.enforcer.AddPolicy(roleKey, tenantKey, perm); err != nil {
			return err
		}
	}

	// 如果最终没有任何权限，添加占位权限
	if len(permissions) == 0 {
		return m.enforcer.AddPolicy(roleKey, tenantKey, core.Permission{Resource: core.ResourcePlaceholder, Action: core.ActionNone})
	}

	return nil
}

// isRoleExists 检查角色是否存在
func (m *roleManager) isRoleExists(roleKey string) (bool, error) {
	// 优先从数据库检查角色是否存在
	exists, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// 如果数据库中不存在，检查是否有权限策略（兼容旧数据）
	policies, err := m.enforcer.GetPolicies(roleKey, "")
	if err != nil {
		return false, err
	}

	if len(policies) > 0 {
		return true, nil
	}

	// 检查是否有用户分配
	return m.enforcer.IsRoleInUse(roleKey)
}

// getCustomRolesByTenant 获取指定租户的自定义角色列表
// tenantKey: 目标租户键，通常从JWT token解析获得
// 返回: 指定租户的角色 + 全局角色(Domain="*")
func (m *roleManager) getCustomRolesByTenant(tenantKey string) ([]*core.Role, error) {
	// 获取所有权限策略
	policies, err := m.enforcer.GetAllPolicies()
	if err != nil {
		return nil, err
	}

	// 收集角色权限（按租户过滤）
	rolePermsByTenant := make(map[string][]core.Permission)
	roleTenantMap := make(map[string]string) // 记录角色的归属租户

	for _, policy := range policies {
		roleKey := policy.Subject

		// 租户过滤逻辑
		if tenantKey != "" {
			// 指定租户：只包含该租户和全局角色(*)
			// 例如：tenantKey="tenant1" 时，返回 Domain="tenant1" 和 Domain="*" 的角色
			if policy.Domain != tenantKey && policy.Domain != "*" {
				continue
			}
		}
		// 如果tenantKey为空，返回所有角色（管理员视图）

		// 跳过占位权限
		if policy.Resource == core.ResourcePlaceholder && policy.Action == core.ActionNone {
			// 确保角色存在于map中，即使没有实际权限
			if _, exists := rolePermsByTenant[roleKey]; !exists {
				rolePermsByTenant[roleKey] = []core.Permission{}
				roleTenantMap[roleKey] = policy.Domain
			}
			continue
		}

		rolePermsByTenant[roleKey] = append(rolePermsByTenant[roleKey], core.Permission{
			Resource: policy.Resource,
			Action:   policy.Action,
		})

		// 记录角色的归属租户（优先记录非全局域）
		if _, exists := roleTenantMap[roleKey]; !exists || policy.Domain != "*" {
			roleTenantMap[roleKey] = policy.Domain
		}
	}

	// 转换为Role结构
	var roles []*core.Role
	for roleKey, permissions := range rolePermsByTenant {
		roles = append(roles, &core.Role{
			Key:         roleKey,
			Name:        roleKey,
			Description: "自定义角色",
			Permissions: permissions,
			TenantKey:   roleTenantMap[roleKey],
		})
	}

	return roles, nil
}

// matchRoleFilter 检查角色是否匹配过滤条件
func (m *roleManager) matchRoleFilter(role *core.Role, filter *core.RoleFilter) bool {
	if filter == nil {
		return true
	}

	if filter.KeyPattern != "" && role.Key != filter.KeyPattern {
		return false
	}

	if filter.NamePattern != "" && role.Name != filter.NamePattern {
		return false
	}

	if filter.TenantKey != "" && role.TenantKey != filter.TenantKey {
		return false
	}

	return true
}

// isSystemPermission 检查是否为系统权限（使用安全验证器配置）
func (m *roleManager) isSystemPermission(permission core.Permission) bool {
	return m.securityValidator.GetPermissionType(permission) == core.PermissionTypeSystem
}

// createRoleMetadata 在数据库中创建角色元数据
func (m *roleManager) createRoleMetadata(roleKey, name, description, tenantKey, createdBy string) error {
	insertSQL := `
		INSERT INTO roles (role_key, name, description, tenant_key, created_by)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := m.dbConn.Exec(insertSQL, roleKey, name, description, tenantKey, createdBy)
	return err
}

// updateRoleMetadata 更新数据库中的角色元数据
func (m *roleManager) updateRoleMetadata(roleKey, name, description string) error {
	updateSQL := `
		UPDATE roles
		SET name = $2, description = $3, updated_at = CURRENT_TIMESTAMP
		WHERE role_key = $1
	`
	_, err := m.dbConn.Exec(updateSQL, roleKey, name, description)
	return err
}

// deleteRoleMetadata 删除数据库中的角色元数据
func (m *roleManager) deleteRoleMetadata(roleKey string) error {
	deleteSQL := `DELETE FROM roles WHERE role_key = $1`
	_, err := m.dbConn.Exec(deleteSQL, roleKey)
	return err
}

// getRoleMetadata 从数据库获取角色元数据
func (m *roleManager) getRoleMetadata(roleKey string) (*roleMetadata, error) {
	var role roleMetadata
	selectSQL := `
		SELECT role_key, name, description, tenant_key, created_at, updated_at, created_by
		FROM roles WHERE role_key = $1
	`
	err := m.dbConn.QueryRow(&role, selectSQL, roleKey)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// listRoleMetadata 从数据库获取角色列表
func (m *roleManager) listRoleMetadata(tenantKey string) ([]*roleMetadata, error) {
	var roles []*roleMetadata
	var selectSQL string
	var args []interface{}

	if tenantKey == "" {
		// 获取所有角色
		selectSQL = `
			SELECT role_key, name, description, tenant_key, created_at, updated_at, created_by
			FROM roles ORDER BY created_at DESC
		`
	} else {
		// 获取指定租户的角色（包括全局角色）
		selectSQL = `
			SELECT role_key, name, description, tenant_key, created_at, updated_at, created_by
			FROM roles WHERE tenant_key = $1 OR tenant_key = '*'
			ORDER BY created_at DESC
		`
		args = append(args, tenantKey)
	}

	err := m.dbConn.QueryRows(&roles, selectSQL, args...)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// isRoleExistsInDB 检查角色是否在数据库中存在
func (m *roleManager) isRoleExistsInDB(roleKey string) (bool, error) {
	var count int
	countSQL := `SELECT COUNT(*) FROM roles WHERE role_key = $1`
	err := m.dbConn.QueryRow(&count, countSQL, roleKey)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
