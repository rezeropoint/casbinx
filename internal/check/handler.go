package check

import (
	"github.com/rezeropoint/casbinx/core"
)

// checkManager 权限检查管理器实现
type checkManager struct {
	enforcer *core.Enforcer // 核心执行器
}

// newCheckManager 创建权限检查管理器
func newCheckManager(enforcer *core.Enforcer) Manager {
	return &checkManager{
		enforcer: enforcer,
	}
}

// CheckPermission 权限检查 (包括直接权限和通过角色继承的权限)
func (m *checkManager) CheckPermission(userKey, tenantKey string, permission core.Permission) (bool, error) {
	// 使用 Casbin 的 Enforce 方法，它会自动检查用户的直接权限和角色继承权限
	return m.enforcer.CheckPermission(userKey, tenantKey, permission)
}

// HasDirectPermission 检查用户是否有直接权限 (不包括角色权限)
func (m *checkManager) HasDirectPermission(userKey, tenantKey string, permission core.Permission) (bool, error) {
	// 只检查用户的直接权限，不包括通过角色继承的权限
	return m.enforcer.HasDirectPermission(userKey, tenantKey, permission)
}

// HasRole 检查用户是否有角色
func (m *checkManager) HasRole(userKey, roleKey, tenantKey string) (bool, error) {
	// 检查用户在指定租户下是否有指定角色
	roles, err := m.enforcer.GetRolesForUser(userKey, tenantKey)
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		if role == roleKey {
			return true, nil
		}
	}
	return false, nil
}

// CheckMultiplePermissions 批量权限检查
func (m *checkManager) CheckMultiplePermissions(userKey, tenantKey string, permissions []core.Permission) ([]bool, error) {
	results := make([]bool, len(permissions))

	for i, permission := range permissions {
		hasPermission, err := m.CheckPermission(userKey, tenantKey, permission)
		if err != nil {
			return nil, err
		}
		results[i] = hasPermission
	}

	return results, nil
}

// HasAnyPermission 检查是否有任意一个权限
func (m *checkManager) HasAnyPermission(userKey, tenantKey string, permissions []core.Permission) (bool, error) {
	for _, permission := range permissions {
		hasPermission, err := m.CheckPermission(userKey, tenantKey, permission)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions 检查是否有所有权限
func (m *checkManager) HasAllPermissions(userKey, tenantKey string, permissions []core.Permission) (bool, error) {
	for _, permission := range permissions {
		hasPermission, err := m.CheckPermission(userKey, tenantKey, permission)
		if err != nil {
			return false, err
		}
		if !hasPermission {
			return false, nil
		}
	}
	return true, nil
}

// CanAccessResource 检查是否可以访问资源 (任意操作)
func (m *checkManager) CanAccessResource(userKey, tenantKey string, resource core.Resource) (bool, error) {
	// 使用基础操作列表，检查用户是否对资源有任意一种操作权限
	for _, action := range core.AllActions {
		hasPermission, err := m.CheckPermission(userKey, tenantKey, core.Permission{Resource: resource, Action: action})
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}
	return false, nil
}

// GetAvailableActions 获取用户可执行的操作
func (m *checkManager) GetAvailableActions(userKey, tenantKey string, resource core.Resource) ([]core.Action, error) {
	// 获取资源的可用操作列表
	resourceActions := core.GetResourceActions(resource)
	availableActions := make([]core.Action, 0)

	for _, action := range resourceActions {
		hasPermission, err := m.CheckPermission(userKey, tenantKey, core.Permission{Resource: resource, Action: action})
		if err != nil {
			return nil, err
		}
		if hasPermission {
			availableActions = append(availableActions, action)
		}
	}

	return availableActions, nil
}

// CanAccessTenant 检查是否可以访问租户
func (m *checkManager) CanAccessTenant(userKey, tenantKey string) (bool, error) {
	// 1. 检查用户是否有全局租户管理权限
	hasTenantReadPermission, err := m.CheckPermission(userKey, "*", core.Permission{
		Resource: core.ResourceTenant,
		Action:   core.ActionRead,
	})
	if err != nil {
		return false, err
	}
	if hasTenantReadPermission {
		return true, nil // 有全局租户读取权限，可以访问任何租户
	}

	// 2. 检查用户是否在该租户下有任何角色
	roles, err := m.enforcer.GetRolesForUser(userKey, tenantKey)
	if err != nil {
		return false, err
	}

	// 如果有角色，则可以访问该租户
	if len(roles) > 0 {
		return true, nil
	}

	// 3. 检查是否有针对该租户的任何核心资源权限
	// 检查所有核心资源的权限，只要有任意一个权限就可以访问租户
	coreResources := []core.Resource{
		core.ResourceUser,
		core.ResourceRole,
		core.ResourcePermission,
		core.ResourceSystem,
		core.ResourceTenant,
	}

	for _, resource := range coreResources {
		for _, action := range core.AllActions {
			hasPermission, err := m.CheckPermission(userKey, tenantKey, core.Permission{
				Resource: resource,
				Action:   action,
			})
			if err != nil {
				return false, err
			}
			if hasPermission {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetUserTenants 获取用户可访问的租户
func (m *checkManager) GetUserTenants(userKey string) ([]string, error) {
	// 获取用户的所有租户权限
	policies, err := m.enforcer.GetPolicies(userKey, "")
	if err != nil {
		return nil, err
	}

	tenantSet := make(map[string]struct{})

	// 从策略中提取租户
	for _, policy := range policies {
		if policy.Domain != "" && policy.Domain != "*" {
			tenantSet[policy.Domain] = struct{}{}
		}
	}

	// 获取用户的所有角色分配
	groupPolicies, err := m.enforcer.GetGroupingPolicies()
	if err != nil {
		return nil, err
	}

	// 从角色分配中提取租户 (只获取该用户的角色分配)
	for _, policy := range groupPolicies {
		if policy.UserKey == userKey && policy.TenantKey != "" && policy.TenantKey != "*" {
			tenantSet[policy.TenantKey] = struct{}{}
		}
	}

	// 转换为切片
	tenants := make([]string, 0, len(tenantSet))
	for tenant := range tenantSet {
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}
