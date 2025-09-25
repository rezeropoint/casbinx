package user

import (
	"casbinx/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// userManager 用户权限管理器实现，处理用户特有的业务逻辑
type userManager struct {
	enforcer *core.Enforcer
	dbConn   sqlx.SqlConn
}

// newUserManager 创建用户权限管理器实现
func newUserManager(dsn string, enforcer *core.Enforcer) *userManager {
	// 初始化 PostgreSQL - 使用 URL 格式的 DSN
	dbConn := sqlx.NewSqlConn("postgres", dsn)

	return &userManager{
		enforcer: enforcer,
		dbConn:   dbConn,
	}
}

// GrantPermission 为用户授予权限
func (m *userManager) GrantPermission(operatorKey, userKey, tenantKey string, permission core.Permission) error {
	// 验证参数
	if err := m.validateParams(userKey, permission); err != nil {
		return err
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		return err
	}

	// 安全检查已在engine层处理

	// 调用core层添加权限
	return m.enforcer.AddPolicy(userKey, tenantKey, permission)
}

// RevokePermission 撤销用户权限
func (m *userManager) RevokePermission(operatorKey, userKey, tenantKey string, permission core.Permission) error {
	// 验证参数
	if err := m.validateParams(userKey, permission); err != nil {
		return err
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		return err
	}

	// 安全检查已在engine层处理

	// 调用core层移除权限
	return m.enforcer.RemovePolicy(userKey, tenantKey, permission)
}

// GetDirectPermissions 获取用户直接权限（不包括角色权限）
func (m *userManager) GetDirectPermissions(userKey, tenantKey string) ([]core.Permission, error) {
	if userKey == "" {
		return nil, core.ErrInvalidParameter
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		// 如果是角色，返回空权限列表而不是错误
		return []core.Permission{}, nil
	}

	// 调用core层获取直接权限
	return m.enforcer.GetDirectPermissions(userKey, tenantKey)
}

// GetEffectivePermissions 获取用户有效权限（包括角色继承）
func (m *userManager) GetEffectivePermissions(userKey, tenantKey string) ([]core.Permission, error) {
	if userKey == "" {
		return nil, core.ErrInvalidParameter
	}

	// 获取隐式权限（包括通过角色继承的权限）
	return m.enforcer.GetImplicitPermissions(userKey, tenantKey)
}

// AssignRole 为用户分配角色
func (m *userManager) AssignRole(operatorKey, userKey, roleKey, tenantKey string) error {
	// 验证参数
	if userKey == "" || roleKey == "" {
		return core.ErrInvalidParameter
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		return err
	}

	// 验证角色存在
	if err := m.validateRoleExists(roleKey); err != nil {
		return err
	}

	// 注意：在新的设计中，角色分配不再有独立的安全检查
	// 安全控制通过权限级别来实现，角色只是权限的容器

	// 调用core层分配角色
	return m.enforcer.AddGroupingPolicy(userKey, roleKey, tenantKey)
}

// RemoveRole 移除用户角色
func (m *userManager) RemoveRole(operatorKey, userKey, roleKey, tenantKey string) error {
	// 验证参数
	if userKey == "" || roleKey == "" {
		return core.ErrInvalidParameter
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		return err
	}

	// 注意：在新的设计中，角色移除不再有独立的安全检查
	// 安全控制通过权限级别来实现

	// 调用core层移除角色
	return m.enforcer.RemoveGroupingPolicy(userKey, roleKey, tenantKey)
}

// GetUserRoles 获取用户角色
func (m *userManager) GetUserRoles(userKey, tenantKey string) ([]string, error) {
	if userKey == "" {
		return nil, core.ErrInvalidParameter
	}
	// 获取租户域中的角色
	tenantRoles, err := m.enforcer.GetRolesForUser(userKey, tenantKey)
	if err != nil {
		return nil, err
	}

	// 获取全局域（*）中的角色（如超级管理员）
	globalRoles, err := m.enforcer.GetRolesForUser(userKey, "*")
	if err != nil {
		return tenantRoles, nil // 返回租户角色，忽略全局角色错误
	}

	// 合并并去重
	roleSet := make(map[string]bool)
	var roles []string

	for _, role := range tenantRoles {
		if !roleSet[role] {
			roles = append(roles, role)
			roleSet[role] = true
		}
	}

	for _, role := range globalRoles {
		if !roleSet[role] {
			roles = append(roles, role)
			roleSet[role] = true
		}
	}

	return roles, nil
}

// ClearUserPermissions 清除用户的所有直接权限
func (m *userManager) ClearUserPermissions(operatorKey, userKey string) error {
	if userKey == "" {
		return core.ErrInvalidParameter
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		return err
	}

	return m.enforcer.ClearPolicies(userKey)
}

// ClearUserRoles 清除用户的所有角色分配
func (m *userManager) ClearUserRoles(operatorKey, userKey string) error {
	if userKey == "" {
		return core.ErrInvalidParameter
	}

	// 验证userKey不是角色
	if err := m.validateNotRole(userKey); err != nil {
		return err
	}

	// 安全检查已在engine层处理

	return m.enforcer.ClearUserRoles(userKey)
}

// GetUserPermissionsByResource 获取用户在指定资源上的权限
func (m *userManager) GetUserPermissionsByResource(userKey, tenantKey, resource string) ([]core.Permission, error) {
	permissions, err := m.GetEffectivePermissions(userKey, tenantKey)
	if err != nil {
		return nil, err
	}

	return core.FilterPermissions(permissions, core.Resource(resource), ""), nil
}
