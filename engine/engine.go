package engine

import (
	"github.com/rezeropoint/casbinx/core"
)

// CasbinX CasbinX权限管理引擎接口
type CasbinX interface {
	// 用户权限管理
	GrantPermission(operatorKey, userKey, tenantKey string, permission core.Permission) error  // 授予用户权限
	RevokePermission(operatorKey, userKey, tenantKey string, permission core.Permission) error // 撤销用户权限

	// 安全版本的权限查询（需要操作者身份验证）
	GetDirectPermissionsSecure(operatorKey, userKey, tenantKey string) ([]core.Permission, error)    // 安全查询用户直接权限
	GetEffectivePermissionsSecure(operatorKey, userKey, tenantKey string) ([]core.Permission, error) // 安全查询用户有效权限
	ClearUserPermissions(operatorKey, userKey, tenantKey string) error                               // 清除用户在指定租户的所有权限
	GetUserPermissionsByResource(userKey, tenantKey, resource string) ([]core.Permission, error)     // 获取用户对特定资源的权限

	// 用户角色分配
	AssignRole(operatorKey, userKey, roleKey, tenantKey string) error // 为用户分配角色
	RemoveRole(operatorKey, userKey, roleKey, tenantKey string) error // 移除用户角色
	GetUserRoles(userKey, tenantKey string) ([]string, error)         // 获取用户角色列表
	ClearUserRoles(operatorKey, userKey string) error                 // 清除用户所有角色分配

	// 角色管理
	CreateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error // 创建角色
	UpdateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error // 更新角色信息
	DeleteRole(roleKey string) error                                                                               // 删除角色
	GetRole(roleKey string) (*core.Role, error)                                                                    // 获取角色详情
	ListRoles(tenantKey string, filter *core.RoleFilter) ([]*core.Role, error)                                     // 获取角色列表

	// 角色权限管理
	GetRolePermissions(roleKey string) ([]core.Permission, error)                        // 获取角色权限列表
	GrantRolePermission(operatorKey, roleKey string, permission core.Permission) error   // 授予角色权限
	RevokeRolePermission(operatorKey, roleKey string, permission core.Permission) error  // 撤销角色权限
	SetRolePermissions(operatorKey, roleKey string, permissions []core.Permission) error // 设置角色权限(覆盖)

	// 角色用户管理
	GetUsersWithRole(roleKey, tenantKey string) ([]string, error)           // 获取拥有指定角色的用户列表
	GetAllGroupingPolicies(tenantKey string) ([]core.GroupingPolicy, error) // 获取指定租户的所有角色分配

	// 权限检查 (包括用户直接权限和通过角色继承的权限)
	CheckPermission(userKey, tenantKey string, permission core.Permission) (bool, error)     // 检查用户权限(含角色继承)
	HasDirectPermission(userKey, tenantKey string, permission core.Permission) (bool, error) // 检查用户直接权限(不含角色)
	HasRole(userKey, roleKey, tenantKey string) (bool, error)                                // 检查用户是否拥有角色

	// 批量权限检查
	CheckMultiplePermissions(userKey, tenantKey string, permissions []core.Permission) ([]bool, error) // 批量检查权限
	HasAnyPermission(userKey, tenantKey string, permissions []core.Permission) (bool, error)           // 检查是否拥有任意一个权限
	HasAllPermissions(userKey, tenantKey string, permissions []core.Permission) (bool, error)          // 检查是否拥有所有权限

	// 资源和租户访问检查
	CanAccessResource(userKey, tenantKey string, resource core.Resource) (bool, error)            // 检查是否可访问资源(任意操作)
	CanAccessTenant(userKey, tenantKey string) (bool, error)                                      // 检查是否可访问租户
	GetAvailableActions(userKey, tenantKey string, resource core.Resource) ([]core.Action, error) // 获取用户对资源的可用操作
	GetUserTenants(userKey string) ([]string, error)                                              // 获取用户可访问的租户列表

	// 租户初始化
	InitializeTenant(tenantKey, adminUserKey, adminRoleKey string) error // 初始化租户并分配管理员

	// Watcher 管理
	RefreshPolicy() error // 手动刷新策略（从数据库重新加载）
}

// NewCasbinx 创建CasbinX权限管理引擎
func NewCasbinx(c core.Config) (CasbinX, error) {
	return newCasbinxClient(c)
}
