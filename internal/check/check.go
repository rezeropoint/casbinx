package check

import (
	"github.com/rezeropoint/casbinx/core"
)

// Manager 权限检查管理器接口
type Manager interface {
	// 基础权限检查
	CheckPermission(userKey, tenantKey string, permission core.Permission) (bool, error)     // 检查用户权限(含角色继承)
	HasDirectPermission(userKey, tenantKey string, permission core.Permission) (bool, error) // 检查用户直接权限(不含角色)

	// 角色检查
	HasRole(userKey, roleKey, tenantKey string) (bool, error) // 检查用户是否拥有角色

	// 批量权限检查
	CheckMultiplePermissions(userKey, tenantKey string, permissions []core.Permission) ([]bool, error) // 批量检查权限
	HasAnyPermission(userKey, tenantKey string, permissions []core.Permission) (bool, error)           // 检查是否拥有任意一个权限
	HasAllPermissions(userKey, tenantKey string, permissions []core.Permission) (bool, error)          // 检查是否拥有所有权限

	// 资源级别检查
	CanAccessResource(userKey, tenantKey string, resource core.Resource) (bool, error)            // 检查是否可访问资源(任意操作)
	GetAvailableActions(userKey, tenantKey string, resource core.Resource) ([]core.Action, error) // 获取用户对资源的可用操作

	// 租户级别检查
	CanAccessTenant(userKey, tenantKey string) (bool, error) // 检查是否可访问租户
	GetUserTenants(userKey string) ([]string, error)         // 获取用户可访问的租户列表

}

// NewManager 创建权限检查管理器
func NewManager(enforcer *core.Enforcer) Manager {
	return newCheckManager(enforcer)
}
