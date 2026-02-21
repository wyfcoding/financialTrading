// 变更说明：新增 feemanagement 领域事件。
package domain

import (
	"time"
	"github.com/shopspring/decimal"
)

const (
	EventFeeRuleCreated   = "fee.rule.created"
	EventFeeCalculated    = "fee.calculated"
	EventRebateProcessed  = "fee.rebate.processed"
)

type FeeEvent struct {
	EventID   string          `json:"event_id"`
	TradeID   string          `json:"trade_id"`
	AccountID string          `json:"account_id"`
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency"`
	Timestamp time.Time       `json:"timestamp"`
}
