package core

import (
	"fmt"
)

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	CheckPermission(userKey, tenantKey string, permission Permission) (bool, error)
}

// SecurityValidator 安全验证器
type SecurityValidator struct {
	config            SecurityConfig
	permissionChecker PermissionChecker
}

// NewSecurityValidator 创建安全验证器
func NewSecurityValidator(config SecurityConfig) *SecurityValidator {
	return &SecurityValidator{
		config:            config,
		permissionChecker: nil, // 延迟设置
	}
}

// SetPermissionChecker 设置权限检查器（解决循环依赖）
func (sv *SecurityValidator) SetPermissionChecker(checker PermissionChecker) {
	sv.permissionChecker = checker
}

// ValidatePermissionGrant 验证权限授予操作
func (sv *SecurityValidator) ValidatePermissionGrant(operatorKey, targetUserKey, tenantKey string, permission Permission) error {
	// 1. 防止自我提权检查（优先检查，覆盖所有其他检查）
	if err := sv.PreventSelfElevation(operatorKey, targetUserKey, permission); err != nil {
		return err
	}

	// 2. 检查是否为系统权限
	if sv.isSystemPermission(permission) {
		return ErrSystemPermissionImmutable
	}

	// 3. 验证操作者权限 - 使用正确的租户域进行权限验证
	if err := sv.validateOperatorPermission(operatorKey, tenantKey, permission); err != nil {
		return err
	}

	return nil
}

// ValidatePermissionGrantWithDomain 验证权限授予操作（支持租户域）
func (sv *SecurityValidator) ValidatePermissionGrantWithDomain(operatorKey, targetUserKey, operatorDomain string, permission Permission) error {
	// 1. 防止自我提权检查（优先检查，覆盖所有其他检查）
	if err := sv.PreventSelfElevation(operatorKey, targetUserKey, permission); err != nil {
		return err
	}

	// 2. 检查是否为系统权限
	if sv.isSystemPermission(permission) {
		return ErrSystemPermissionImmutable
	}

	// 3. 验证操作者权限
	if err := sv.validateOperatorPermission(operatorKey, operatorDomain, permission); err != nil {
		return err
	}

	return nil
}

// ValidatePermissionRevoke 验证权限撤销操作
func (sv *SecurityValidator) ValidatePermissionRevoke(operatorKey, targetUserKey, tenantKey string, permission Permission) error {
	// 1. 防止自我提权检查（撤销时也要检查，防止通过撤销再重新授予绕过限制）
	if err := sv.PreventSelfElevation(operatorKey, targetUserKey, permission); err != nil {
		return err
	}

	// 2. 检查是否为系统权限
	if sv.isSystemPermission(permission) {
		return ErrSystemPermissionImmutable
	}

	// 3. 验证操作者权限 - 使用正确的租户域进行权限验证
	if err := sv.validateOperatorPermission(operatorKey, tenantKey, permission); err != nil {
		return err
	}

	return nil
}

// isSystemPermission 检查是否为系统权限
func (sv *SecurityValidator) isSystemPermission(permission Permission) bool {
	for _, sysPerm := range sv.config.SystemPermissions {
		if permission.Resource == sysPerm.Resource && permission.Action == sysPerm.Action {
			return true
		}
	}
	return false
}

// GetPermissionType 获取权限类型
func (sv *SecurityValidator) GetPermissionType(permission Permission) PermissionType {
	if sv.isSystemPermission(permission) {
		return PermissionTypeSystem
	}

	return PermissionTypeNormal
}

// CanGrantPermission 检查操作者是否可以授予指定权限
func (sv *SecurityValidator) CanGrantPermission(operatorKey string, permission Permission) bool {
	permType := sv.GetPermissionType(permission)

	switch permType {
	case PermissionTypeSystem:
		return false // 系统权限不能被授予
	case PermissionTypeNormal:
		return true // 普通权限可以自由授予
	}

	return false
}

// GetSecurityConfig 获取安全配置
func (sv *SecurityValidator) GetSecurityConfig() SecurityConfig {
	return sv.config
}

// UpdateSecurityConfig 更新安全配置
func (sv *SecurityValidator) UpdateSecurityConfig(config SecurityConfig) error {
	if err := ValidateSecurityConfig(config); err != nil {
		return err
	}
	sv.config = config
	return nil
}

// ValidateSecurityConfig 验证安全配置的有效性
func ValidateSecurityConfig(config SecurityConfig) error {

	// 验证系统权限格式
	for _, perm := range config.SystemPermissions {
		if !perm.IsValid() {
			return fmt.Errorf("系统权限格式无效: %s:%s", perm.Resource, perm.Action)
		}
	}

	return nil
}

// PreventSelfElevation 防止自我提权
func (sv *SecurityValidator) PreventSelfElevation(operatorKey, targetKey string, permission Permission) error {
	// 如果禁用了防自我提权，直接返回
	if !sv.config.PreventSelfElevation {
		return nil
	}

	// 如果操作者和目标用户不是同一人，不需要检查
	if operatorKey != targetKey {
		return nil
	}

	// 检查是否是管理权限（权限管理、用户管理、角色管理等）
	// 注意：系统权限也被认为是管理权限，因为它们涉及敏感操作
	if sv.isManagementPermission(permission) {
		return ErrSelfElevationPrevented
	}

	return nil
}

// isManagementPermission 检查是否为管理权限
func (sv *SecurityValidator) isManagementPermission(permission Permission) bool {
	// 权限管理权限
	if permission.Resource == ResourcePermission && (permission.Action == ActionWrite || permission.Action == ActionDelete) {
		return true
	}

	// 用户管理权限
	if permission.Resource == ResourceUser && (permission.Action == ActionWrite || permission.Action == ActionDelete) {
		return true
	}

	// 角色管理权限
	if permission.Resource == ResourceRole && (permission.Action == ActionWrite || permission.Action == ActionDelete) {
		return true
	}

	// 系统权限都是管理权限
	if sv.isSystemPermission(permission) {
		return true
	}

	return false
}

// validateOperatorPermission 验证操作者是否有权限执行指定操作
func (sv *SecurityValidator) validateOperatorPermission(operatorKey, domain string, permission Permission) error {
	if sv.permissionChecker == nil {
		// 如果没有权限检查器，跳过操作者权限验证（兼容模式）
		return nil
	}

	permType := sv.GetPermissionType(permission)

	switch permType {
	case PermissionTypeSystem:
		// 系统权限需要权限管理权限（需要write权限）
		permissionToCheck := Permission{Resource: ResourcePermission, Action: ActionWrite}
		hasPermission, err := sv.permissionChecker.CheckPermission(operatorKey, domain, permissionToCheck)
		if err != nil {
			return fmt.Errorf("检查操作者权限时出错: %w", err)
		}
		if !hasPermission {
			return fmt.Errorf("操作者 %s 没有权限管理权限，无法操作系统权限 %s:%s", operatorKey, permission.Resource, permission.Action)
		}

	case PermissionTypeNormal:
		// 普通权限需要权限管理权限（需要write权限）
		permissionToCheck := Permission{Resource: ResourcePermission, Action: ActionWrite}
		hasPermission, err := sv.permissionChecker.CheckPermission(operatorKey, domain, permissionToCheck)
		if err != nil {
			return fmt.Errorf("检查操作者权限时出错: %w", err)
		}
		if !hasPermission {
			return fmt.Errorf("操作者 %s 没有权限管理权限，无法操作权限 %s:%s", operatorKey, permission.Resource, permission.Action)
		}
	}

	return nil
}
