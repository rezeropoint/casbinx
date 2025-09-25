package user

import (
	"casbinx/core"
	"fmt"
)

// validateParams 验证基本参数
func (m *userManager) validateParams(userKey string, permission core.Permission) error {
	if userKey == "" || permission.Resource == "" || permission.Action == "" {
		return core.ErrInvalidParameter
	}
	return nil
}

// validateNotRole 验证主体不是角色
func (m *userManager) validateNotRole(subject string) error {
	// 首先检查是否在 roles 表中存在（最准确的方法）
	isRole, err := m.isRoleExistsInDB(subject)
	if err != nil {
		return err
	}

	if isRole {
		return fmt.Errorf("不能直接给角色 '%s' 操作权限，请使用角色管理接口", subject)
	}

	// 兼容性检查：检查是否在角色分配策略中作为角色使用（处理旧数据）
	isInUse, err := m.enforcer.IsRoleInUse(subject)
	if err != nil {
		return err
	}

	if isInUse {
		return fmt.Errorf("不能直接给角色 '%s' 操作权限，请使用角色管理接口", subject)
	}

	return nil
}

// validateRoleExists 验证角色存在
func (m *userManager) validateRoleExists(roleKey string) error {
	// 首先检查是否在 roles 表中存在（最准确的方法）
	isRole, err := m.isRoleExistsInDB(roleKey)
	if err != nil {
		return err
	}

	if isRole {
		return nil
	}

	// 兼容性检查：检查自定义角色是否存在（通过检查是否有权限策略或被分配，处理旧数据）
	policies, err := m.enforcer.GetPolicies(roleKey, "")
	if err != nil {
		return err
	}

	if len(policies) > 0 {
		return nil
	}

	// 检查是否有用户被分配了该角色
	isInUse, err := m.enforcer.IsRoleInUse(roleKey)
	if err != nil {
		return err
	}

	if isInUse {
		return nil
	}

	return fmt.Errorf("角色 '%s' 不存在", roleKey)
}

// isRoleExistsInDB 检查角色是否在数据库中存在
func (m *userManager) isRoleExistsInDB(roleKey string) (bool, error) {
	var count int
	countSQL := `SELECT COUNT(*) FROM roles WHERE role_key = $1`
	err := m.dbConn.QueryRow(&count, countSQL, roleKey)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
