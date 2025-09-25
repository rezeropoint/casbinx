package role

import (
	"github.com/rezeropoint/casbinx/core"
)

// Manager 角色权限管理器接口
type Manager interface {
	// 角色管理
	CreateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error // 创建角色
	UpdateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error // 更新角色信息
	DeleteRole(roleKey string) error                                                                               // 删除角色
	GetRole(roleKey string) (*core.Role, error)                                                                    // 获取角色详情
	ListRoles(tenantKey string, filter *core.RoleFilter) ([]*core.Role, error)                                     // 获取角色列表

	// 角色系统权限检查
	HasSystemPermissions(roleKey string) (bool, error)                             // 检查角色是否包含系统权限
	UserRoleHasSystemPermissions(userKey, roleKey, tenantKey string) (bool, error) // 检查用户的角色是否包含系统权限

	// 角色权限管理
	GetRolePermissions(roleKey string) ([]core.Permission, error)                   // 获取角色权限列表
	GrantPermission(operatorKey, roleKey string, permission core.Permission) error  // 授予角色权限
	RevokePermission(operatorKey, roleKey string, permission core.Permission) error // 撤销角色权限
	SetRolePermissions(roleKey string, permissions []core.Permission) error         // 设置角色权限(覆盖)

	// 角色用户管理
	GetUsersWithRole(roleKey, tenantKey string) ([]string, error)           // 获取拥有指定角色的用户列表
	GetAllGroupingPolicies(tenantKey string) ([]core.GroupingPolicy, error) // 获取指定租户的所有角色分配
}

// NewManager 创建角色权限管理器
func NewManager(dsn string, enforcer *core.Enforcer, securityValidator *core.SecurityValidator) (Manager, error) {
	return newRoleManager(dsn, enforcer, securityValidator)
}
