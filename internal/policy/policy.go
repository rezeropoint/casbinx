package policy

import (
	"github.com/rezeropoint/casbinx/core"
)

// Manager 策略管理器接口
type Manager interface {
	// RefreshPolicy 手动刷新策略（从数据库重新加载）
	RefreshPolicy() error
}

// NewManager 创建策略管理器
func NewManager(enforcer *core.Enforcer) (Manager, error) {
	return newPolicyManager(enforcer)
}
