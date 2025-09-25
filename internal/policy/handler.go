package policy

import (
	"fmt"

	"casbinx/core"
)

// policyManager 策略管理器实现
type policyManager struct {
	enforcer *core.Enforcer
}

// newPolicyManager 创建策略管理器实现
func newPolicyManager(enforcer *core.Enforcer) (*policyManager, error) {
	if enforcer == nil {
		return nil, fmt.Errorf("核心执行器未初始化")
	}
	return &policyManager{
		enforcer: enforcer,
	}, nil
}

// RefreshPolicy 手动刷新策略（从数据库重新加载）
func (p *policyManager) RefreshPolicy() error {
	if p.enforcer == nil {
		return fmt.Errorf("核心执行器未初始化")
	}

	err := p.enforcer.LoadPolicy()
	if err != nil {
		return fmt.Errorf("刷新策略失败: %v", err)
	}

	return nil
}
