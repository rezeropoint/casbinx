package core

import (
	"fmt"
	"strings"
)

// PermissionFilter 权限过滤器
type PermissionFilter struct {
	UserKey   string   `json:"userKey"`   // 用户标识过滤条件
	TenantKey string   `json:"tenantKey"` // 租户标识过滤条件
	Resource  Resource `json:"resource"`  // 资源类型过滤条件
	Action    Action   `json:"action"`    // 操作类型过滤条件
}

// PermissionDefinition 权限定义
type PermissionDefinition struct {
	Resource    Resource `json:"resource"`    // 资源类型标识
	Action      Action   `json:"action"`      // 操作类型标识
	Description string   `json:"description"` // 权限描述说明
	Category    string   `json:"category"`    // 权限分类标签
}

// GetPermissionKey 获取权限键
func (pd PermissionDefinition) GetPermissionKey() string {
	return fmt.Sprintf("%s:%s", pd.Resource, pd.Action)
}

// IsValid 检查权限定义是否有效
func (pd PermissionDefinition) IsValid() bool {
	return pd.Resource != "" && pd.Action != ""
}

// PermissionType 权限类型
type PermissionType int

const (
	PermissionTypeNormal PermissionType = 0 // 普通权限：可自由授予和撤销
	PermissionTypeSystem PermissionType = 1 // 系统权限：不可变更（immutable）
)

// Permission 权限结构体
type Permission struct {
	Resource Resource `json:"resource"` // 资源类型，如user、graph、infoatom等
	Action   Action   `json:"action"`   // 操作类型，如read、write、delete等
}

// String 返回权限的字符串表示
func (p Permission) String() string {
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

// IsValid 检查权限是否有效
func (p Permission) IsValid() bool {
	return p.Resource != "" && p.Action != ""
}

// Equal 检查两个权限是否相等
func (p Permission) Equal(other Permission) bool {
	return p.Resource == other.Resource && p.Action == other.Action
}

// IsEmpty 检查权限是否为空
func (p Permission) IsEmpty() bool {
	return p.Resource == "" || p.Action == ""
}

// ParsePermission 解析权限字符串 "resource:action"
func ParsePermission(permStr string) (Permission, error) {
	parts := strings.Split(permStr, ":")
	if len(parts) != 2 {
		return Permission{}, ErrInvalidParameter
	}

	action, err := ParseAction(strings.TrimSpace(parts[1]))
	if err != nil {
		return Permission{}, err
	}

	return Permission{
		Resource: Resource(strings.TrimSpace(parts[0])),
		Action:   action,
	}, nil
}

// MergePermissions 合并权限列表，去重
func MergePermissions(permLists ...[]Permission) []Permission {
	permMap := make(map[string]Permission)

	for _, permissions := range permLists {
		for _, perm := range permissions {
			key := perm.String()
			permMap[key] = perm
		}
	}

	result := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		result = append(result, perm)
	}

	return result
}

// ContainsPermission 检查权限列表是否包含指定权限
func ContainsPermission(permissions []Permission, target Permission) bool {
	for _, perm := range permissions {
		if perm.Resource == target.Resource && perm.Action == target.Action {
			return true
		}
	}
	return false
}

// FilterPermissions 根据条件过滤权限
func FilterPermissions(permissions []Permission, resource Resource, action Action) []Permission {
	var result []Permission

	for _, perm := range permissions {
		match := true

		if resource != "" && perm.Resource != resource {
			match = false
		}

		if action != "" && perm.Action != action {
			match = false
		}

		if match {
			result = append(result, perm)
		}
	}

	return result
}
