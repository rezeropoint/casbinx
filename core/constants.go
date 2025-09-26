package core

import "fmt"

// Action 权限操作类型 - 使用结构体实现类型安全
type Action string
type Resource string

// ParseAction 从字符串解析 Action（仅用于数据库读取等场景）
func ParseAction(s string) (Action, error) {
	switch s {
	case "read":
		return ActionRead, nil
	case "write":
		return ActionWrite, nil
	case "delete":
		return ActionDelete, nil
	case "_none":
		return ActionNone, nil
	default:
		return Action(s), fmt.Errorf("invalid action: %s, only 'read', 'write', 'delete', '_none' are supported", s)
	}
}

func ParseSystemResource(s string) (Resource, error) {
	switch s {
	case "tenant":
		return ResourceTenant, nil
	case "system":
		return ResourceSystem, nil
	case "user":
		return ResourceUser, nil
	case "permission":
		return ResourcePermission, nil
	case "role":
		return ResourceRole, nil
	case "organization":
		return ResourceOrganization, nil
	case "tag_user":
		return ResourceTagUser, nil
	case "tag_tenant":
		return ResourceTagTenant, nil
	case "placeholder":
		return ResourcePlaceholder, nil
	default:
		return Resource(s), fmt.Errorf("invalid system resource: %s, only 'tenant', 'system', 'user', 'permission', 'role', 'tag_user', 'tag_tenant', 'placeholder' are supported", s)
	}
}

// 基础权限操作常量 - 仅支持三种基本操作
var (
	ActionRead   = Action("read")   // 读取/查看权限
	ActionWrite  = Action("write")  // 创建/更新/配置权限
	ActionDelete = Action("delete") // 删除权限
	ActionNone   = Action("_none")  // 占位符，用于角色标识
)

// 基础资源常量（仅用于系统核心功能）
const (
	ResourceTenant       = Resource("tenant")     // 租户资源
	ResourceSystem       = Resource("system")     // 系统资源
	ResourceUser         = Resource("user")       // 用户资源
	ResourcePermission   = Resource("permission") // 权限资源
	ResourceRole         = Resource("role")       // 角色资源
	ResourceOrganization = Resource("organization")
	ResourceTagUser      = Resource("tag_user")     // 用户标签资源
	ResourceTagTenant    = Resource("tag_tenant")   // 租户标签资源
	ResourcePlaceholder  = Resource("_placeholder") // 占位符，用于角色标识
)

// AllActions 基础权限操作列表 - 仅包含三种基本操作
var AllActions = []Action{
	ActionRead,
	ActionWrite,
	ActionDelete,
}

// DefaultResourceActions 默认资源与可用操作的映射（仅用于系统核心资源）
var DefaultResourceActions = map[Resource][]Action{
	ResourceTenant:       {ActionRead, ActionWrite, ActionDelete},
	ResourceSystem:       {ActionRead, ActionWrite, ActionDelete},
	ResourceUser:         {ActionRead, ActionWrite, ActionDelete},
	ResourcePermission:   {ActionRead, ActionWrite, ActionDelete},
	ResourceRole:         {ActionRead, ActionWrite, ActionDelete},
	ResourceOrganization: {ActionRead, ActionWrite, ActionDelete},
	ResourceTagUser:      {ActionRead, ActionWrite, ActionDelete},
	ResourceTagTenant:    {ActionRead, ActionWrite, ActionDelete},
}

// GetResourceActions 获取指定资源的可用操作列表
func GetResourceActions(resource Resource) []Action {
	if actions, exists := DefaultResourceActions[resource]; exists {
		return actions
	}
	// 默认返回基础操作
	return AllActions
}

// HasManagePermission 检查是否有管理权限（read + write + delete）
func HasManagePermission(checkFunc func(resource Resource, action Action) (bool, error), resource Resource) (bool, error) {
	// 管理权限需要同时拥有读、写、删除权限
	for _, action := range AllActions {
		hasPermission, err := checkFunc(resource, action)
		if err != nil {
			return false, err
		}
		if !hasPermission {
			return false, nil
		}
	}
	return true, nil
}
