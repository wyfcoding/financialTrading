// 生成摘要：从 creditrisk 合并到 risk 域。信用风险是风控子域。
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// CreditExposure 信用敞口监控。
// 专门计算对手方风险（Counterparty Risk）、敞口(Exposure)限额。
type CreditExposure struct {
	gorm.Model
	// CounterpartyID 对手方机构或用户 ID。
	CounterpartyID string `gorm:"type:varchar(64);uniqueIndex" json:"counterparty_id"`
	// CreditLimit 授信总额。
	CreditLimit decimal.Decimal `gorm:"type:decimal(36,4);comment:授信总额" json:"credit_limit"`
	// UsedExposure 已占用敞口。
	UsedExposure decimal.Decimal `gorm:"type:decimal(36,4);comment:已占用敞口" json:"used_exposure"`
	// MarginHeld 被扣押的保证金。
	MarginHeld decimal.Decimal `gorm:"type:decimal(36,4);comment:被扣押的保证金" json:"margin_held"`
	// Status 信用状态：HEALTHY, MARGIN_CALL, DEFAULTED。
	Status string `gorm:"type:varchar(16);default:'HEALTHY'" json:"status"`
}

// AvailableCredit 可用信用额度。
func (c *CreditExposure) AvailableCredit() decimal.Decimal {
	return c.CreditLimit.Sub(c.UsedExposure).Add(c.MarginHeld)
}

// OperationalEvent 操作风险事件（胖手指防范）。
// 当后台风控指标或员工行为异常时阻断并报警。
type OperationalEvent struct {
	gorm.Model
	// EventCode 事件编码，全局唯一。
	EventCode string `gorm:"type:varchar(64);uniqueIndex" json:"event_code"`
	// Category 事件类别：FAT_FINGER, SYSTEM_OUTAGE 等。
	Category string `gorm:"type:varchar(32);comment:例如 FAT_FINGER, SYSTEM_OUTAGE" json:"category"`
	// Severity 严重级别。
	Severity string `gorm:"type:varchar(16)" json:"severity"`
	// OperatorID 操作人 ID。
	OperatorID uint64 `gorm:"index" json:"operator_id"`
	// Description 事件描述。
	Description string `gorm:"type:text" json:"description"`
	// LossAmount 估算资损金额。
	LossAmount float64 `gorm:"type:decimal(36,4);comment:如果造成资损的估算金额" json:"loss_amount"`
	// IsResolved 是否已解决。
	IsResolved bool `gorm:"default:false" json:"is_resolved"`
}
