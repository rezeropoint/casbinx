package engine

import (
	"fmt"
	"log"
	"os"

	"github.com/rezeropoint/casbinx/core"
	"github.com/rezeropoint/casbinx/internal/check"
	"github.com/rezeropoint/casbinx/internal/policy"
	"github.com/rezeropoint/casbinx/internal/role"
	"github.com/rezeropoint/casbinx/internal/user"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// casbinxClient casbinx客户端实现
type casbinxClient struct {
	userManager       user.Manager            // 用户权限管理器
	roleManager       role.Manager            // 角色权限管理器
	checkManager      check.Manager           // 权限检查管理器
	securityValidator *core.SecurityValidator // 安全验证器
	policyManager     policy.Manager          // 策略管理器
}

// newCasbinxClient 创建casbinx客户端
func newCasbinxClient(c core.Config) (*casbinxClient, error) {
	// 如果未配置安全设置，使用默认安全配置
	securityConfig := c.Security
	if len(securityConfig.SystemPermissions) == 0 {
		securityConfig = core.DefaultSecurityConfig()
	}

	// 验证 Watcher 配置（强制要求 Redis）
	watcherConfig := c.Watcher
	if watcherConfig.Redis.Addr == "" {
		return nil, fmt.Errorf("config.Watcher.Redis.Addr 未设置")
	}
	gormDB, err := gorm.Open(postgres.Open(c.Dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("GORM 数据库连接失败: %v", err)
	}

	// 设置默认路径
	modelPaths := c.PossiblePaths
	if len(modelPaths) == 0 {
		modelPaths = []string{
			"etc/casbin_model.conf",
			"./casbin_model.conf",
			"../etc/casbin_model.conf",
		}
	}

	var modelPath string
	for _, path := range modelPaths {
		if _, err := os.Stat(path); err == nil {
			modelPath = path
			break
		}
	}

	if modelPath == "" {
		return nil, fmt.Errorf("Casbin模型文件不存在，已尝试路径: %v", modelPaths)
	}

	// 创建适配器
	adapter, err := gormadapter.NewAdapterByDBUseTableName(gormDB, "", "casbin_rules")
	if err != nil {
		return nil, fmt.Errorf("创建Casbin适配器失败: %v", err)
	}

	// 创建Casbin执行器
	casbinEnforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("创建Casbin执行器失败: %v", err)
	}

	// 启用自动保存和日志
	casbinEnforcer.EnableAutoSave(true)
	casbinEnforcer.EnableLog(true)

	// 创建和配置 Redis Watcher
	watcher, err := rediswatcher.NewWatcher(watcherConfig.Redis.Addr, rediswatcher.WatcherOptions{
		Options: redis.Options{
			Network:  watcherConfig.Redis.Network,
			Password: watcherConfig.Redis.Password,
			DB:       watcherConfig.Redis.DB,
		},
		Channel:    watcherConfig.Redis.Channel,
		IgnoreSelf: watcherConfig.Redis.IgnoreSelf,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Redis Watcher 失败: %v", err)
	}

	// 设置 Watcher 到 Casbin 执行器
	err = casbinEnforcer.SetWatcher(watcher)
	if err != nil {
		return nil, fmt.Errorf("设置 Watcher 失败: %v", err)
	}

	// 设置更新回调，当收到策略变更通知时自动重新加载策略
	err = watcher.SetUpdateCallback(func(msg string) {
		err := casbinEnforcer.LoadPolicy()
		if err != nil {
			log.Printf("[CasbinX] 重新加载策略失败: %v", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("设置 Watcher 更新回调失败: %v", err)
	}

	// 启用自动通知 Watcher（当本实例修改策略时自动通知其他实例）
	casbinEnforcer.EnableAutoNotifyWatcher(true)

	// 创建核心执行器
	coreEnforcer, err := core.NewEnforcer(casbinEnforcer)
	if err != nil {
		return nil, fmt.Errorf("创建核心执行器失败: %v", err)
	}

	// 创建安全验证器
	securityValidator := core.NewSecurityValidator(securityConfig)

	// 创建管理器
	userManager := user.NewManager(c.Dsn, coreEnforcer)
	checkManager := check.NewManager(coreEnforcer)
	roleManager, err := role.NewManager(c.Dsn, coreEnforcer, securityValidator)
	if err != nil {
		return nil, err
	}
	policyManager, err := policy.NewManager(coreEnforcer)
	if err != nil {
		return nil, fmt.Errorf("创建策略管理器失败: %v", err)
	}

	// 设置权限检查器解决循环依赖
	securityValidator.SetPermissionChecker(checkManager)

	return &casbinxClient{
		userManager:       userManager,
		roleManager:       roleManager,
		checkManager:      checkManager,
		securityValidator: securityValidator,
		policyManager:     policyManager,
	}, nil
}

// 用户权限管理方法实现
func (c *casbinxClient) GrantPermission(operatorKey, userKey, tenantKey string, permission core.Permission) error {
	// 安全检查：进行提权验证
	if err := c.securityValidator.ValidatePermissionGrant(operatorKey, userKey, tenantKey, permission); err != nil {
		return err
	}

	return c.userManager.GrantPermission(operatorKey, userKey, tenantKey, permission)
}

func (c *casbinxClient) RevokePermission(operatorKey, userKey, tenantKey string, permission core.Permission) error {
	// 安全检查：进行权限撤销验证
	if err := c.securityValidator.ValidatePermissionRevoke(operatorKey, userKey, tenantKey, permission); err != nil {
		return err
	}

	return c.userManager.RevokePermission(operatorKey, userKey, tenantKey, permission)
}

// GetDirectPermissionsSecure 安全地获取用户直接权限（需要权限验证）
func (c *casbinxClient) GetDirectPermissionsSecure(operatorKey, userKey, tenantKey string) ([]core.Permission, error) {
	// 安全检查：验证查询权限
	if err := c.validateQueryPermission(operatorKey, userKey, tenantKey); err != nil {
		return nil, err
	}

	return c.userManager.GetDirectPermissions(userKey, tenantKey)
}

// GetEffectivePermissionsSecure 安全地获取用户有效权限（需要权限验证）
func (c *casbinxClient) GetEffectivePermissionsSecure(operatorKey, userKey, tenantKey string) ([]core.Permission, error) {
	// 安全检查：验证查询权限
	if err := c.validateQueryPermission(operatorKey, userKey, tenantKey); err != nil {
		return nil, err
	}

	return c.userManager.GetEffectivePermissions(userKey, tenantKey)
}

// validateQueryPermission 验证查询权限
func (c *casbinxClient) validateQueryPermission(operatorKey, targetUserKey, tenantKey string) error {
	// 用户可以查询自己的权限
	if operatorKey == targetUserKey {
		return nil
	}

	// 检查操作者在指定租户内是否有用户查看权限（只需要read权限即可查询）
	permission := core.Permission{Resource: core.ResourceUser, Action: core.ActionRead}
	hasPermission, err := c.checkManager.CheckPermission(operatorKey, tenantKey, permission)
	if err != nil {
		return fmt.Errorf("检查操作者权限时出错: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("操作者 %s 在租户 %s 中没有用户查看权限，无法查询用户 %s 的权限信息", operatorKey, tenantKey, targetUserKey)
	}

	return nil
}

func (c *casbinxClient) ClearUserPermissions(operatorKey, userKey, tenantKey string) error {
	// 安全检查：验证操作者是否有用户管理权限（清除权限是管理操作）
	// 先检查全局域权限
	globalCheckFunc := func(resource core.Resource, action core.Action) (bool, error) {
		permission := core.Permission{Resource: resource, Action: action}
		return c.checkManager.CheckPermission(operatorKey, "*", permission)
	}
	hasGlobalPermission, err := core.HasManagePermission(globalCheckFunc, core.ResourceUser)
	if err != nil {
		return fmt.Errorf("检查操作者全局权限时出错: %w", err)
	}

	// 如果没有全局权限，检查指定租户的权限
	hasTenantPermission := false
	if !hasGlobalPermission {
		tenantCheckFunc := func(resource core.Resource, action core.Action) (bool, error) {
			permission := core.Permission{Resource: resource, Action: action}
			return c.checkManager.CheckPermission(operatorKey, tenantKey, permission)
		}
		hasTenantPermission, err = core.HasManagePermission(tenantCheckFunc, core.ResourceUser)
		if err != nil {
			return fmt.Errorf("检查操作者租户权限时出错: %w", err)
		}
	}

	if !hasGlobalPermission && !hasTenantPermission {
		return fmt.Errorf("操作者 %s 没有在租户 %s 中的用户管理权限，无法清除用户权限", operatorKey, tenantKey)
	}

	return c.userManager.ClearUserPermissions(operatorKey, userKey)
}

func (c *casbinxClient) GetUserPermissionsByResource(userKey, tenantKey, resource string) ([]core.Permission, error) {
	return c.userManager.GetUserPermissionsByResource(userKey, tenantKey, resource)
}

func (c *casbinxClient) AssignRole(operatorKey, userKey, roleKey, tenantKey string) error {
	// 安全检查：验证操作者是否有用户管理权限
	// 验证操作者有用户管理权限
	userPermission := core.Permission{Resource: core.ResourceUser, Action: core.ActionWrite}
	hasUserPermission, err := c.checkManager.CheckPermission(operatorKey, tenantKey, userPermission)
	if err != nil {
		return fmt.Errorf("检查操作者用户管理权限时出错: %w", err)
	}
	if !hasUserPermission {
		return fmt.Errorf("操作者 %s 没有用户管理权限，无法分配角色", operatorKey)
	}

	// 验证操作者有角色管理权限
	rolePermission := core.Permission{Resource: core.ResourceRole, Action: core.ActionWrite}
	hasRolePermission, err := c.checkManager.CheckPermission(operatorKey, tenantKey, rolePermission)
	if err != nil {
		return fmt.Errorf("检查操作者角色管理权限时出错: %w", err)
	}
	if !hasRolePermission {
		return fmt.Errorf("操作者 %s 没有角色管理权限，无法分配角色", operatorKey)
	}

	// 检查角色是否包含系统权限
	hasSystemPerms, err := c.roleManager.HasSystemPermissions(roleKey)
	if err != nil {
		return fmt.Errorf("检查角色系统权限时出错: %w", err)
	}

	if hasSystemPerms {
		// 系统角色只能通过租户初始化接口分配，普通角色分配接口不允许
		return core.ErrSystemRoleAssignmentDenied
	}

	return c.userManager.AssignRole(operatorKey, userKey, roleKey, tenantKey)
}

func (c *casbinxClient) RemoveRole(operatorKey, userKey, roleKey, tenantKey string) error {
	// 安全检查：验证操作者是否有用户管理权限
	// 验证操作者有用户管理权限
	userPermission := core.Permission{Resource: core.ResourceUser, Action: core.ActionWrite}
	hasUserPermission, err := c.checkManager.CheckPermission(operatorKey, tenantKey, userPermission)
	if err != nil {
		return fmt.Errorf("检查操作者用户管理权限时出错: %w", err)
	}
	if !hasUserPermission {
		return fmt.Errorf("操作者 %s 没有用户管理权限，无法分配角色", operatorKey)
	}

	// 验证操作者有角色管理权限
	rolePermission := core.Permission{Resource: core.ResourceRole, Action: core.ActionWrite}
	hasRolePermission, err := c.checkManager.CheckPermission(operatorKey, tenantKey, rolePermission)
	if err != nil {
		return fmt.Errorf("检查操作者角色管理权限时出错: %w", err)
	}
	if !hasRolePermission {
		return fmt.Errorf("操作者 %s 没有角色管理权限，无法分配角色", operatorKey)
	}

	// 检查用户的角色是否包含系统权限
	hasSystemPerms, err := c.roleManager.UserRoleHasSystemPermissions(userKey, roleKey, tenantKey)
	if err != nil {
		return fmt.Errorf("检查用户角色系统权限时出错: %w", err)
	}

	if hasSystemPerms {
		// 系统角色只能通过租户初始化接口分配，不能移除
		return core.ErrSystemRoleRemovalDenied
	}

	return c.userManager.RemoveRole(operatorKey, userKey, roleKey, tenantKey)
}

func (c *casbinxClient) GetUserRoles(userKey, tenantKey string) ([]string, error) {
	return c.userManager.GetUserRoles(userKey, tenantKey)
}

func (c *casbinxClient) ClearUserRoles(operatorKey, userKey string) error {
	return c.userManager.ClearUserRoles(operatorKey, userKey)
}

func (c *casbinxClient) HasDirectPermission(userKey, tenantKey string, permission core.Permission) (bool, error) {
	return c.checkManager.HasDirectPermission(userKey, tenantKey, permission)
}

func (c *casbinxClient) HasRole(userKey, roleKey, tenantKey string) (bool, error) {
	return c.checkManager.HasRole(userKey, roleKey, tenantKey)
}

func (c *casbinxClient) CheckMultiplePermissions(userKey, tenantKey string, permissions []core.Permission) ([]bool, error) {
	return c.checkManager.CheckMultiplePermissions(userKey, tenantKey, permissions)
}

func (c *casbinxClient) HasAnyPermission(userKey, tenantKey string, permissions []core.Permission) (bool, error) {
	return c.checkManager.HasAnyPermission(userKey, tenantKey, permissions)
}

func (c *casbinxClient) HasAllPermissions(userKey, tenantKey string, permissions []core.Permission) (bool, error) {
	return c.checkManager.HasAllPermissions(userKey, tenantKey, permissions)
}

func (c *casbinxClient) CanAccessResource(userKey, tenantKey string, resource core.Resource) (bool, error) {
	return c.checkManager.CanAccessResource(userKey, tenantKey, resource)
}

func (c *casbinxClient) CanAccessTenant(userKey, tenantKey string) (bool, error) {
	return c.checkManager.CanAccessTenant(userKey, tenantKey)
}

func (c *casbinxClient) GetAvailableActions(userKey, tenantKey string, resource core.Resource) ([]core.Action, error) {
	return c.checkManager.GetAvailableActions(userKey, tenantKey, resource)
}

func (c *casbinxClient) GetUserTenants(userKey string) ([]string, error) {
	return c.checkManager.GetUserTenants(userKey)
}

// 角色权限管理方法实现
func (c *casbinxClient) CreateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error {
	// 安全检查：验证角色中的权限
	for _, permission := range permissions {
		// 使用新的带域验证方法
		if err := c.securityValidator.ValidatePermissionGrantWithDomain(operatorKey, roleKey, tenantKey, permission); err != nil {
			return fmt.Errorf("角色权限验证失败 %s:%s - %w", permission.Resource, permission.Action, err)
		}
	}

	return c.roleManager.CreateRole(operatorKey, roleKey, roleName, description, tenantKey, permissions)
}

func (c *casbinxClient) UpdateRole(operatorKey, roleKey, roleName, description, tenantKey string, permissions []core.Permission) error {
	// 检查全局角色操作权限
	if err := c.validateGlobalRoleOperation(operatorKey, roleKey); err != nil {
		return err
	}

	// 获取角色的旧权限
	oldPermissions, err := c.roleManager.GetRolePermissions(roleKey)
	if err != nil {
		return fmt.Errorf("获取角色旧权限失败: %w", err)
	}

	// 找出新增和删除的权限
	addedPermissions := findAddedPermissions(oldPermissions, permissions)
	removedPermissions := findRemovedPermissions(oldPermissions, permissions)

	// 安全检查：只验证新增的权限
	for _, permission := range addedPermissions {
		if err := c.securityValidator.ValidatePermissionGrant(operatorKey, roleKey, tenantKey, permission); err != nil {
			return fmt.Errorf("新增角色权限验证失败 %s:%s - %w", permission.Resource, permission.Action, err)
		}
	}

	// 安全检查：只验证删除的权限（防止删除系统权限）
	for _, permission := range removedPermissions {
		if err := c.securityValidator.ValidatePermissionRevoke(operatorKey, roleKey, tenantKey, permission); err != nil {
			return fmt.Errorf("删除角色权限验证失败 %s:%s - %w", permission.Resource, permission.Action, err)
		}
	}

	return c.roleManager.UpdateRole(operatorKey, roleKey, roleName, description, tenantKey, permissions)
}

func (c *casbinxClient) DeleteRole(roleKey string) error {
	return c.roleManager.DeleteRole(roleKey)
}

func (c *casbinxClient) GetRole(roleKey string) (*core.Role, error) {
	return c.roleManager.GetRole(roleKey)
}

func (c *casbinxClient) ListRoles(tenantKey string, filter *core.RoleFilter) ([]*core.Role, error) {
	return c.roleManager.ListRoles(tenantKey, filter)
}

func (c *casbinxClient) GetRolePermissions(roleKey string) ([]core.Permission, error) {
	return c.roleManager.GetRolePermissions(roleKey)
}

func (c *casbinxClient) GrantRolePermission(operatorKey, roleKey string, permission core.Permission) error {
	// 检查全局角色操作权限
	if err := c.validateGlobalRoleOperation(operatorKey, roleKey); err != nil {
		return err
	}

	// 获取角色信息确定其租户域
	role, err := c.roleManager.GetRole(roleKey)
	if err != nil {
		return fmt.Errorf("获取角色信息失败: %w", err)
	}

	// 使用角色所属的租户域（不会为空，默认为"*"表示全局角色）
	roleTenantKey := role.TenantKey

	// 安全检查：验证权限授予
	permissionToValidate := core.Permission{Resource: permission.Resource, Action: permission.Action}
	if err := c.securityValidator.ValidatePermissionGrant(operatorKey, roleKey, roleTenantKey, permissionToValidate); err != nil {
		return err
	}

	return c.roleManager.GrantPermission(operatorKey, roleKey, permission)
}

func (c *casbinxClient) RevokeRolePermission(operatorKey, roleKey string, permission core.Permission) error {
	// 检查全局角色操作权限
	if err := c.validateGlobalRoleOperation(operatorKey, roleKey); err != nil {
		return err
	}

	// 获取角色信息确定其租户域
	role, err := c.roleManager.GetRole(roleKey)
	if err != nil {
		return fmt.Errorf("获取角色信息失败: %w", err)
	}

	// 使用角色所属的租户域（不会为空，默认为"*"表示全局角色）
	roleTenantKey := role.TenantKey

	// 安全检查：验证权限撤销
	permissionToValidate := core.Permission{Resource: permission.Resource, Action: permission.Action}
	if err := c.securityValidator.ValidatePermissionRevoke(operatorKey, roleKey, roleTenantKey, permissionToValidate); err != nil {
		return err
	}

	return c.roleManager.RevokePermission(operatorKey, roleKey, permission)
}

func (c *casbinxClient) SetRolePermissions(operatorKey, roleKey string, permissions []core.Permission) error {
	// 检查全局角色操作权限
	if err := c.validateGlobalRoleOperation(operatorKey, roleKey); err != nil {
		return err
	}

	// 获取角色信息确定其租户域
	role, err := c.roleManager.GetRole(roleKey)
	if err != nil {
		return fmt.Errorf("获取角色信息失败: %w", err)
	}

	// 使用角色所属的租户域（不会为空，默认为"*"表示全局角色）
	roleTenantKey := role.TenantKey

	// 获取角色的旧权限
	oldPermissions := role.Permissions

	// 找出新增和删除的权限
	addedPermissions := findAddedPermissions(oldPermissions, permissions)
	removedPermissions := findRemovedPermissions(oldPermissions, permissions)

	// 安全检查：只验证新增的权限
	for _, permission := range addedPermissions {
		if err := c.securityValidator.ValidatePermissionGrant(operatorKey, roleKey, roleTenantKey, permission); err != nil {
			return fmt.Errorf("新增角色权限验证失败 %s:%s - %w", permission.Resource, permission.Action, err)
		}
	}

	// 安全检查：只验证删除的权限（防止删除系统权限）
	for _, permission := range removedPermissions {
		if err := c.securityValidator.ValidatePermissionRevoke(operatorKey, roleKey, roleTenantKey, permission); err != nil {
			return fmt.Errorf("删除角色权限验证失败 %s:%s - %w", permission.Resource, permission.Action, err)
		}
	}

	return c.roleManager.SetRolePermissions(roleKey, permissions)
}

func (c *casbinxClient) GetUsersWithRole(roleKey, tenantKey string) ([]string, error) {
	return c.roleManager.GetUsersWithRole(roleKey, tenantKey)
}

func (c *casbinxClient) GetAllGroupingPolicies(tenantKey string) ([]core.GroupingPolicy, error) {
	return c.roleManager.GetAllGroupingPolicies(tenantKey)
}

// CheckPermission 权限检查快捷方法
func (c *casbinxClient) CheckPermission(userKey, tenantKey string, permission core.Permission) (bool, error) {
	return c.checkManager.CheckPermission(userKey, tenantKey, permission)
}

// InitializeTenant 初始化租户并分配管理员
func (c *casbinxClient) InitializeTenant(tenantKey, adminUserKey, adminRoleKey string) error {

	// 该接口是为了确保系统权限被限制时，在初始化租户的场景仍然能分配系统权限

	// 验证参数
	if tenantKey == "" || adminUserKey == "" || adminRoleKey == "" {
		return core.ErrInvalidParameter
	}

	// 1. 检查角色是否存在
	role, err := c.GetRole(adminRoleKey)
	if err != nil {
		return fmt.Errorf("指定的管理员角色 '%s' 不存在", adminRoleKey)
	}

	// 2. 检查角色权限是否符合租户管理员要求
	hasTenantPermissions := false
	hasSystemBasePermissions := false

	for _, perm := range role.Permissions {
		// 检查是否包含租户管理权限（不允许）
		if perm.Resource == core.ResourceTenant || perm.Resource == core.ResourceTagTenant {
			hasTenantPermissions = true
		}
		// 检查是否包含系统基础权限（必须）
		// 系统基础权限包括：system, user, permission, role, tag_user，但不包括 tenant 和 tag_tenant
		if perm.Resource == core.ResourceSystem ||
			perm.Resource == core.ResourceUser ||
			perm.Resource == core.ResourcePermission ||
			perm.Resource == core.ResourceRole ||
			perm.Resource == core.ResourceTagUser {
			hasSystemBasePermissions = true
		}
	}

	// 不允许有租户管理权限
	if hasTenantPermissions {
		return fmt.Errorf("角色 '%s' 包含租户管理权限，租户内管理员不允许跨租户操作", adminRoleKey)
	}

	// 必须有系统权限
	if !hasSystemBasePermissions {
		return fmt.Errorf("角色 '%s' 缺少系统级权限，无法作为租户管理员角色", adminRoleKey)
	}

	// 3. 分配角色给管理员用户（绕过系统权限检查）
	return c.userManager.AssignRole("system", adminUserKey, adminRoleKey, tenantKey)
}

// hasGlobalRoleAssignments 检查角色是否有全局域分配
func (c *casbinxClient) hasGlobalRoleAssignments(roleKey string) (bool, error) {
	// 获取在全局域分配该角色的用户
	users, err := c.roleManager.GetUsersWithRole(roleKey, "*")
	if err != nil {
		return false, err
	}

	// 如果有用户在全局域被分配了这个角色，则该角色有全局域分配
	return len(users) > 0, nil
}

// hasGlobalPermission 检查用户是否有全局权限
func (c *casbinxClient) hasGlobalPermission(operatorKey string, permission core.Permission) (bool, error) {
	return c.checkManager.CheckPermission(operatorKey, "*", permission)
}

// validateGlobalRoleOperation 验证全局角色操作权限
func (c *casbinxClient) validateGlobalRoleOperation(operatorKey, roleKey string) error {
	// 检查角色是否有全局域分配
	hasGlobalAssignments, err := c.hasGlobalRoleAssignments(roleKey)
	if err != nil {
		return fmt.Errorf("检查角色全局分配失败: %w", err)
	}

	if !hasGlobalAssignments {
		// 角色没有全局域分配，无需特殊检查
		return nil
	}

	// 检查操作者是否有全局权限（角色管理权限）
	hasGlobalPerm, err := c.hasGlobalPermission(operatorKey, core.Permission{
		Resource: core.ResourceRole,
		Action:   core.ActionWrite,
	})
	if err != nil {
		return fmt.Errorf("检查全局权限失败: %w", err)
	}

	if !hasGlobalPerm {
		return core.ErrGlobalRoleAccessDenied
	}

	return nil
}

// isTenantInitializationScenario 判断是否为租户初始化场景
func (c *casbinxClient) isTenantInitializationScenario(userKey, roleKey, tenantKey string) bool {
	// 租户初始化场景的特征：
	// 1. 用户key包含"admin"且以数字结尾（如admin-001）
	// 2. 角色为"admin"
	// 3. 租户不为空且不为"*"

	if roleKey != "admin" {
		return false
	}

	if tenantKey == "" || tenantKey == "*" {
		return false
	}

	// 检查用户key是否符合admin用户模式
	// 支持格式：admin-001, tenant1-admin-001 等
	return len(userKey) > 5 &&
		(userKey[:5] == "admin" || userKey[len(userKey)-6:] == "-admin" ||
			userKey[len(userKey)-9:len(userKey)-4] == "-admin-")
}

// === Watcher 管理方法实现 ===

// RefreshPolicy 手动刷新策略（从数据库重新加载）
func (c *casbinxClient) RefreshPolicy() error {
	return c.policyManager.RefreshPolicy()
}

// === 权限对比辅助函数 ===

// permissionExists 检查权限是否存在于权限列表中
func permissionExists(permission core.Permission, permissions []core.Permission) bool {
	for _, p := range permissions {
		if p.Resource == permission.Resource && p.Action == permission.Action {
			return true
		}
	}
	return false
}

// findAddedPermissions 找出新增的权限（在新权限中但不在旧权限中）
func findAddedPermissions(oldPermissions, newPermissions []core.Permission) []core.Permission {
	var added []core.Permission
	for _, newPerm := range newPermissions {
		if !permissionExists(newPerm, oldPermissions) {
			added = append(added, newPerm)
		}
	}
	return added
}

// findRemovedPermissions 找出删除的权限（在旧权限中但不在新权限中）
func findRemovedPermissions(oldPermissions, newPermissions []core.Permission) []core.Permission {
	var removed []core.Permission
	for _, oldPerm := range oldPermissions {
		if !permissionExists(oldPerm, newPermissions) {
			removed = append(removed, oldPerm)
		}
	}
	return removed
}
