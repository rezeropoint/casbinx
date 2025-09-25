package core

import (
	"fmt"
	"time"
)

// PolicyType 策略类型
type PolicyType string

const (
	PolicyTypePermission PolicyType = "p" // 权限策略
	PolicyTypeGrouping   PolicyType = "g" // 角色策略
)

// Policy 策略结构体
type Policy struct {
	Type     PolicyType `json:"type"`     // 策略类型，p为权限策略，g为角色分组策略
	Subject  string     `json:"subject"`  // 主体标识，用户ID或角色名
	Domain   string     `json:"domain"`   // 域标识，租户ID或*表示全局
	Resource Resource   `json:"resource"` // 资源类型，如user、graph等
	Action   Action     `json:"action"`   // 操作类型，如read、write等
}

// GroupingPolicy 角色分配策略
type GroupingPolicy struct {
	UserKey   string `json:"userKey"`   // 用户标识
	RoleKey   string `json:"roleKey"`   // 角色标识
	TenantKey string `json:"tenantKey"` // 租户标识，*表示全局角色
}

// PermissionChange 权限变更记录
type PermissionChange struct {
	ID          string    `json:"id"`          // 变更记录唯一标识
	UserKey     string    `json:"userKey"`     // 被操作的用户标识
	Action      Action    `json:"action"`      // 操作类型：grant/revoke/assign/remove
	Target      string    `json:"target"`      // 操作目标：permission或role
	TenantKey   string    `json:"tenantKey"`   // 租户标识
	OperatorKey string    `json:"operatorKey"` // 操作者用户标识
	Timestamp   time.Time `json:"timestamp"`   // 操作时间戳
	Reason      string    `json:"reason"`      // 操作原因描述
}

// Error definitions
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Common errors
var (
	ErrPermissionNotFound   = Error{Code: "PERMISSION_NOT_FOUND", Message: "权限不存在"}
	ErrRoleNotFound         = Error{Code: "ROLE_NOT_FOUND", Message: "角色不存在"}
	ErrUserNotFound         = Error{Code: "USER_NOT_FOUND", Message: "用户不存在"}
	ErrPermissionDenied     = Error{Code: "PERMISSION_DENIED", Message: "权限被拒绝"}
	ErrInvalidParameter     = Error{Code: "INVALID_PARAMETER", Message: "无效参数"}
	ErrCasbinNotInitialized = Error{Code: "CASBIN_NOT_INITIALIZED", Message: "Casbin执行器未初始化"}
	ErrRoleAlreadyExists    = Error{Code: "ROLE_ALREADY_EXISTS", Message: "角色已存在"}

	// 安全相关错误
	ErrSelfElevationPrevented     = Error{Code: "SELF_ELEVATION_PREVENTED", Message: "不允许为自己分配管理员权限"}
	ErrSystemPermissionImmutable  = Error{Code: "SYSTEM_PERMISSION_IMMUTABLE", Message: "系统权限不可变更"}
	ErrSystemRoleImmutable        = Error{Code: "SYSTEM_ROLE_IMMUTABLE", Message: "该角色包含系统级权限（如租户管理、系统配置等），不允许修改。请创建新的自定义角色来调整权限"}
	ErrSystemRoleAssignmentDenied = Error{Code: "SYSTEM_ROLE_ASSIGNMENT_DENIED", Message: "角色包含系统级权限，只能在租户初始化时分配给管理员用户"}
	ErrSystemRoleRemovalDenied    = Error{Code: "SYSTEM_ROLE_REMOVAL_DENIED", Message: "无法移除该角色：角色包含系统级权限，移除后用户将无法管理系统"}
	ErrTenantRoleInvalid          = Error{Code: "TENANT_ROLE_INVALID", Message: "指定的角色包含跨租户权限，不适合作为租户内管理员角色。租户内管理员应使用不包含租户管理权限的角色"}
	ErrGlobalRoleAccessDenied     = Error{Code: "GLOBAL_ROLE_ACCESS_DENIED", Message: "操作全局域角色需要全局权限，当前用户只有租户级权限"}
	ErrDelegationDepthExceeded    = Error{Code: "DELEGATION_DEPTH_EXCEEDED", Message: "超过权限传递深度限制"}
	ErrInvalidPermissionType      = Error{Code: "INVALID_PERMISSION_TYPE", Message: "无效的权限类型"}
)
