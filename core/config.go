package core

// Config CasbinX配置
type Config struct {
	Dsn           string         `json:"dsn"`           // 数据库连接字符串
	PossiblePaths []string       `json:"possiblePaths"` // Casbin模型文件可能的路径
	Security      SecurityConfig `json:"security"`      // 安全配置
	Watcher       WatcherConfig  `json:"watcher"`       // Watcher配置（多副本同步）
}

// SecurityConfig 安全相关配置
type SecurityConfig struct {
	// PreventSelfElevation 防止自我提权
	// true: 禁止用户给自己分配管理权限（默认）
	// false: 允许自我提权（不推荐，仅用于特殊场景）
	PreventSelfElevation bool `json:"preventSelfElevation"`

	// SystemPermissions 系统权限列表（不可授予也不可撤销）
	SystemPermissions []Permission `json:"systemPermissions"`
}

// WatcherConfig Watcher配置
type WatcherConfig struct {
	// Redis 配置（CasbinX 强制使用 Redis Watcher）
	Redis RedisWatcherConfig `json:"redis"`
}

// RedisWatcherConfig Redis Watcher配置
type RedisWatcherConfig struct {
	Network    string `json:"network"`    // Network 网络类型，通常为 "tcp"
	Addr       string `json:"addr"`       // Addr Redis地址，格式：host:port
	Password   string `json:"password"`   // Password Redis密码（可选）
	DB         int    `json:"db"`         // DB Redis数据库编号
	Channel    string `json:"channel"`    // Channel 用于通知的Redis频道
	IgnoreSelf bool   `json:"ignoreSelf"` // IgnoreSelf 是否忽略自己发布的消息
}

// DefaultSecurityConfig 返回默认安全配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		PreventSelfElevation: true, // 默认启用防自我提权

		// 系统权限：完全不可变更
		SystemPermissions: []Permission{
			// 租户管理权限
			{Resource: ResourceTenant, Action: ActionWrite},  // 租户创建/更新权限不可撤销
			{Resource: ResourceTenant, Action: ActionDelete}, // 租户删除权限不可撤销
			{Resource: ResourceTenant, Action: ActionRead},   // 租户读取权限不可撤销

			// 系统管理权限
			{Resource: ResourceSystem, Action: ActionWrite},  // 系统配置权限不可撤销
			{Resource: ResourceSystem, Action: ActionRead},   // 系统读取权限不可撤销
			{Resource: ResourceSystem, Action: ActionDelete}, // 系统删除权限不可撤销

			// 用户管理权限
			{Resource: ResourceUser, Action: ActionWrite}, // 用户创建/更新权限不可撤销
			// {Resource: ResourceUser, Action: ActionRead},   // 用户读取权限不可撤销
			{Resource: ResourceUser, Action: ActionDelete}, // 用户删除权限不可撤销

			// 权限管理权限
			{Resource: ResourcePermission, Action: ActionWrite}, // 权限创建/更新权限不可撤销
			// {Resource: ResourcePermission, Action: ActionRead},   // 权限读取权限不可撤销
			{Resource: ResourcePermission, Action: ActionDelete}, // 权限删除权限不可撤销
		},
	}
}
