package core

// RoleFilter 角色过滤器
type RoleFilter struct {
	KeyPattern  string `json:"keyPattern"`  // 角色键匹配模式
	NamePattern string `json:"namePattern"` // 角色名匹配模式
	TenantKey   string `json:"tenantKey"`   // 租户键过滤条件
}

// Role 角色结构体
type Role struct {
	Key         string       `json:"key"`         // 角色唯一标识符
	Name        string       `json:"name"`        // 角色显示名称
	Description string       `json:"description"` // 角色描述信息
	Permissions []Permission `json:"permissions"` // 角色拥有的权限列表
	TenantKey   string       `json:"tenantKey"`   // 角色归属的租户键，空表示全局角色
}

func (r *Role) GetKey() string  { return r.Key }  // GetKey 获取角色键
func (r *Role) GetName() string { return r.Name } // GetName 获取角色名

// HasPermission 检查角色是否包含指定权限
func (r *Role) HasPermission(resource Resource, action Action) bool {
	for _, perm := range r.Permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
	}
	return false
}

// AddPermission 添加权限到角色
func (r *Role) AddPermission(permission Permission) {
	if !r.HasPermission(permission.Resource, permission.Action) {
		r.Permissions = append(r.Permissions, permission)
	}
}

// RemovePermission 从角色移除权限
func (r *Role) RemovePermission(resource Resource, action Action) {
	for i, perm := range r.Permissions {
		if perm.Resource == resource && perm.Action == action {
			r.Permissions = append(r.Permissions[:i], r.Permissions[i+1:]...)
			break
		}
	}
}

// IsValid 检查角色是否有效
func (r *Role) IsValid() bool {
	return r.Key != "" && r.Name != ""
}

// Clone 克隆角色
func (r *Role) Clone() *Role {
	permissions := make([]Permission, len(r.Permissions))
	copy(permissions, r.Permissions)

	return &Role{
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		Permissions: permissions,
		TenantKey:   r.TenantKey,
	}
}
