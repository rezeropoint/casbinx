package core

import (
	"github.com/casbin/casbin/v2"
)

// Enforcer Casbin执行器的基础封装，提供核心权限操作
type Enforcer struct {
	enforcer *casbin.Enforcer
}

// NewEnforcer 创建核心权限执行器
func NewEnforcer(casbinEnforcer *casbin.Enforcer) (*Enforcer, error) {
	if casbinEnforcer == nil {
		return nil, ErrCasbinNotInitialized
	}

	return &Enforcer{
		enforcer: casbinEnforcer,
	}, nil
}

// === 基础策略操作 ===

// AddPolicy 添加权限策略
func (e *Enforcer) AddPolicy(subject, domain string, permission Permission) error {
	_, err := e.enforcer.AddPolicy(subject, domain, string(permission.Resource), string(permission.Action))
	return err
}

// RemovePolicy 移除权限策略
func (e *Enforcer) RemovePolicy(subject, domain string, permission Permission) error {
	_, err := e.enforcer.RemovePolicy(subject, domain, string(permission.Resource), string(permission.Action))
	return err
}

// GetPolicies 获取指定主体的权限策略
func (e *Enforcer) GetPolicies(subject, domain string) ([]Policy, error) {

	allPolicies, err := e.enforcer.GetPolicy()
	if err != nil {
		return nil, err
	}

	var policies []Policy
	for _, policy := range allPolicies {
		if len(policy) >= 4 {
			action, err := ParseAction(policy[3])
			if err != nil {
				return nil, err
			}
			// 匹配主体和域
			if (subject == "" || policy[0] == subject) &&
				(domain == "" || policy[1] == domain) {
				policies = append(policies, Policy{
					Type:     PolicyTypePermission,
					Subject:  policy[0],
					Domain:   policy[1],
					Resource: Resource(policy[2]),
					Action:   action,
				})
			}
		}
	}

	return policies, nil
}

// GetAllPolicies 获取所有权限策略
func (e *Enforcer) GetAllPolicies() ([]Policy, error) {
	return e.GetPolicies("", "")
}

// ClearPolicies 清除指定主体的所有权限策略
func (e *Enforcer) ClearPolicies(subject string) error {

	policies, err := e.GetPolicies(subject, "")
	if err != nil {
		return err
	}

	for _, policy := range policies {
		_, err := e.enforcer.RemovePolicy(policy.Subject, policy.Domain, string(policy.Resource), string(policy.Action))
		if err != nil {
			return err
		}
	}

	return nil
}

// === 角色分配操作 ===

// AddGroupingPolicy 为用户分配角色
func (e *Enforcer) AddGroupingPolicy(userKey, roleKey, domain string) error {
	_, err := e.enforcer.AddRoleForUserInDomain(userKey, roleKey, domain)
	return err
}

// RemoveGroupingPolicy 移除用户角色
func (e *Enforcer) RemoveGroupingPolicy(userKey, roleKey, domain string) error {
	_, err := e.enforcer.DeleteRoleForUserInDomain(userKey, roleKey, domain)
	return err
}

// GetRolesForUser 获取用户在指定域中的角色
func (e *Enforcer) GetRolesForUser(userKey, domain string) ([]string, error) {
	return e.enforcer.GetRolesForUserInDomain(userKey, domain), nil
}

// ClearUserRoles 清除指定用户的所有角色分配
func (e *Enforcer) ClearUserRoles(userKey string) error {
	// 获取所有角色分配策略
	allGroupPolicies, err := e.enforcer.GetGroupingPolicy()
	if err != nil {
		return err
	}

	// 找到所有该用户的角色分配并移除
	for _, policy := range allGroupPolicies {
		if len(policy) >= 3 && policy[0] == userKey {
			_, err := e.enforcer.DeleteRoleForUserInDomain(policy[0], policy[1], policy[2])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetGroupingPolicies 获取所有角色分配策略
func (e *Enforcer) GetGroupingPolicies() ([]GroupingPolicy, error) {

	allGroupPolicies, err := e.enforcer.GetGroupingPolicy()
	if err != nil {
		return nil, err
	}

	var policies []GroupingPolicy
	for _, policy := range allGroupPolicies {
		if len(policy) >= 3 {
			policies = append(policies, GroupingPolicy{
				UserKey:   policy[0],
				RoleKey:   policy[1],
				TenantKey: policy[2],
			})
		}
	}

	return policies, nil
}

// === 权限检查操作 ===

// CheckPermission 检查权限
func (e *Enforcer) CheckPermission(subject, domain string, permission Permission) (bool, error) {

	// 使用我们的跨域权限继承逻辑，而不是直接使用 Casbin Enforce
	// 获取用户所有有效权限（包括跨域角色继承）
	userPermissions, err := e.GetImplicitPermissions(subject, domain)
	if err != nil {
		return false, err
	}

	// 检查目标权限是否在用户的有效权限中
	for _, userPerm := range userPermissions {
		if userPerm.Resource == permission.Resource && userPerm.Action == permission.Action {
			return true, nil
		}
	}

	return false, nil
}

// GetImplicitPermissions 获取隐式权限（包括角色继承）
func (e *Enforcer) GetImplicitPermissions(userKey, domain string) ([]Permission, error) {

	var allPolicies [][]string

	// 1. 获取用户在指定域的直接权限
	userPolicies, err := e.enforcer.GetPermissionsForUser(userKey, domain)
	if err == nil {
		allPolicies = append(allPolicies, userPolicies...)
	}

	// 2. 获取用户角色（检查指定域和全局域）
	var allRoles []string

	// 2a. 获取用户在指定域的角色
	tenantRoles := e.enforcer.GetRolesForUserInDomain(userKey, domain)
	allRoles = append(allRoles, tenantRoles...)

	// 2b. 获取用户在全局域的角色（如超级管理员）
	if domain != "*" {
		globalRoles := e.enforcer.GetRolesForUserInDomain(userKey, "*")
		allRoles = append(allRoles, globalRoles...)
	}

	// 去重角色
	roleMap := make(map[string]bool)
	var uniqueRoles []string
	for _, role := range allRoles {
		if !roleMap[role] {
			uniqueRoles = append(uniqueRoles, role)
			roleMap[role] = true
		}
	}

	// 3. 获取每个角色的权限
	// 需要考虑角色可能来自不同域，权限也可能定义在不同域
	domainsToCheck := []string{"*"} // 总是检查全局域
	if domain != "*" {
		domainsToCheck = append(domainsToCheck, domain) // 如果不是全局域，也检查指定域
	}

	for _, role := range uniqueRoles {
		// 在所有相关域中查找角色权限
		for _, checkDomain := range domainsToCheck {
			rolePolicies, err := e.enforcer.GetPermissionsForUser(role, checkDomain)
			if err == nil {
				allPolicies = append(allPolicies, rolePolicies...)
			}
		}
	}

	// 4. 转换为 Permission 结构
	permissions := make([]Permission, 0, len(allPolicies))
	for _, policy := range allPolicies {
		if len(policy) >= 4 {
			action, err := ParseAction(policy[3])
			if err != nil {
				return nil, err
			}
			permissions = append(permissions, Permission{
				Resource: Resource(policy[2]),
				Action:   action,
			})
		}
	}

	return permissions, nil
}

// GetDirectPermissions 获取用户的直接权限（不包括角色继承）
func (e *Enforcer) GetDirectPermissions(userKey, domain string) ([]Permission, error) {

	// 获取所有策略，然后过滤出用户的直接权限（不包含角色权限）
	allPolicies, err := e.enforcer.GetPolicy()
	if err != nil {
		return nil, err
	}

	var permissions []Permission
	for _, policy := range allPolicies {
		if len(policy) >= 4 {
			// 匹配主体和域，确保是用户的直接权限（不是角色权限）
			if policy[0] == userKey && policy[1] == domain {
				action, err := ParseAction(policy[3])
				if err != nil {
					return nil, err
				}
				permissions = append(permissions, Permission{
					Resource: Resource(policy[2]),
					Action:   action,
				})
			}
		}
	}

	return permissions, nil
}

// === 工具方法 ===

// IsRoleAssigned 检查角色是否已分配给用户
func (e *Enforcer) IsRoleAssigned(userKey, roleKey, domain string) (bool, error) {
	roles, err := e.GetRolesForUser(userKey, domain)
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

// HasDirectPermission 检查是否有直接权限
func (e *Enforcer) HasDirectPermission(subject, domain string, permission Permission) (bool, error) {
	policies, err := e.GetPolicies(subject, domain)
	if err != nil {
		return false, err
	}

	for _, policy := range policies {
		if policy.Resource == permission.Resource && policy.Action == permission.Action {
			return true, nil
		}
	}

	return false, nil
}

// IsRoleInUse 检查角色是否被使用（有用户分配了该角色）
func (e *Enforcer) IsRoleInUse(roleKey string) (bool, error) {
	groupPolicies, err := e.GetGroupingPolicies()
	if err != nil {
		return false, err
	}

	for _, policy := range groupPolicies {
		if policy.RoleKey == roleKey {
			return true, nil
		}
	}

	return false, nil
}

// GetUsersWithRole 获取拥有指定角色的所有用户
func (e *Enforcer) GetUsersWithRole(roleKey, domain string) ([]string, error) {

	groupPolicies, err := e.GetGroupingPolicies()
	if err != nil {
		return nil, err
	}

	var users []string
	for _, policy := range groupPolicies {
		if policy.RoleKey == roleKey && (domain == "" || policy.TenantKey == domain) {
			users = append(users, policy.UserKey)
		}
	}

	return users, nil
}

// === Watcher 管理方法 ===

// LoadPolicy 手动重新加载策略（用于Watcher同步）
func (e *Enforcer) LoadPolicy() error { return e.enforcer.LoadPolicy() }
