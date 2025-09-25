package user

import (
	"github.com/rezeropoint/casbinx/core"
)

// Manager 用户权限管理器接口
type Manager interface {
	// 权限管理
	GrantPermission(operatorKey, userKey, tenantKey string, permission core.Permission) error    // 授予用户权限
	RevokePermission(operatorKey, userKey, tenantKey string, permission core.Permission) error   // 撤销用户权限
	GetDirectPermissions(userKey, tenantKey string) ([]core.Permission, error)                   // 获取用户直接权限
	GetEffectivePermissions(userKey, tenantKey string) ([]core.Permission, error)                // 获取用户有效权限(含角色继承)
	ClearUserPermissions(operatorKey, userKey string) error                                      // 清除用户所有权限
	GetUserPermissionsByResource(userKey, tenantKey, resource string) ([]core.Permission, error) // 获取用户对特定资源的权限

	// 角色分配
	AssignRole(operatorKey, userKey, roleKey, tenantKey string) error // 为用户分配角色
	RemoveRole(operatorKey, userKey, roleKey, tenantKey string) error // 移除用户角色
	GetUserRoles(userKey, tenantKey string) ([]string, error)         // 获取用户角色列表
	ClearUserRoles(operatorKey, userKey string) error                 // 清除用户所有角色分配

}

// NewManager 创建用户权限管理器
func NewManager(dsn string, enforcer *core.Enforcer) Manager {
	return newUserManager(dsn, enforcer)
}
