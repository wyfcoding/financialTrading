// 变更说明：新增 ledger 领域事件定义。
// 账本服务是金融系统的核心基础设施，所有资金变动必须通过事件流实现完整审计追踪。
// 关键假设：所有事件一经写入不可篡改，采用 Event Sourcing 模式保证数据完整性。
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// EventType 账本领域事件类型常量。
const (
	EventJournalCreated       = "ledger.journal.created"
	EventFundsHeld            = "ledger.funds.held"
	EventFundsReleased        = "ledger.funds.released"
	EventFundsSettled         = "ledger.funds.settled"
	EventBalanceUpdated       = "ledger.balance.updated"
	EventAccountCreated       = "ledger.account.created"
	EventAccountClosed        = "ledger.account.closed"
	EventDayEndReconciled     = "ledger.dayend.reconciled"
	EventAdjustmentPosted     = "ledger.adjustment.posted"
	EventReversalPosted       = "ledger.reversal.posted"
	EventInterestAccrued      = "ledger.interest.accrued"
	EventFeeCharged           = "ledger.fee.charged"
	EventReconciliationFailed = "ledger.reconciliation.failed"
)

// LedgerEvent 账本领域事件基础结构。
// 所有账本事件必须包含事务 ID 以支持跨服务追踪。
type LedgerEvent struct {
	// Metadata 扩展元数据，用于携带业务上下文（如订单号、交易对手等）。
	Metadata map[string]string `json:"metadata,omitempty"`
	// EventID 事件唯一标识。
	EventID string `json:"event_id"`
	// EventType 事件类型标识。
	EventType string `json:"event_type"`
	// AggregateID 聚合根 ID（账户 ID）。
	AggregateID string `json:"aggregate_id"`
	// TransactionID 关联的事务 ID，用于跨服务追踪。
	TransactionID string `json:"transaction_id"`
	// OccurredAt 事件发生时间。
	OccurredAt time.Time `json:"occurred_at"`
	// Version 事件版本号，用于乐观并发控制。
	Version int64 `json:"version"`
}

// JournalCreatedEvent 复式记账凭证创建事件。
// 触发时机：每次资金划转成功后发布。
// 下游消费者：风控服务（实时监控大额转账）、审计服务（记录审计日志）、对账服务。
type JournalCreatedEvent struct {
	LedgerEvent
	// JournalID 凭证唯一标识。
	JournalID string `json:"journal_id"`
	// JournalType 凭证类型（TRANSFER/FEE/INTEREST/ADJUSTMENT/REVERSAL）。
	JournalType string `json:"journal_type"`
	// Entries 分录明细。
	Entries []EntrySnapshot `json:"entries"`
	// TotalAmount 涉及总金额（取借方合计）。
	TotalAmount decimal.Decimal `json:"total_amount"`
	// Currency 币种。
	Currency string `json:"currency"`
	// Description 凭证描述。
	Description string `json:"description"`
}

// EntrySnapshot 分录快照，嵌入事件中用于审计。
type EntrySnapshot struct {
	// EntryID 分录唯一标识。
	EntryID string `json:"entry_id"`
	// AccountID 账户标识。
	AccountID string `json:"account_id"`
	// Direction 借贷方向。
	Direction Direction `json:"direction"`
	// Amount 金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
}

// FundsHeldEvent 资金冻结事件。
// 触发时机：订单下单、保证金冻结等场景。
// 下游消费者：账户服务（更新可用余额）、风控服务（监控冻结比例）。
type FundsHeldEvent struct {
	LedgerEvent
	// AccountID 被冻结的账户。
	AccountID string `json:"account_id"`
	// ReferenceID 业务关联 ID（如订单号）。
	ReferenceID string `json:"reference_id"`
	// ReferenceType 业务关联类型（ORDER/MARGIN/SETTLEMENT）。
	ReferenceType string `json:"reference_type"`
	// Amount 冻结金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
	// ExpiresAt 冻结过期时间，超时自动释放。
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// FundsReleasedEvent 资金解冻事件。
// 触发时机：订单取消、保证金释放等场景。
type FundsReleasedEvent struct {
	LedgerEvent
	// AccountID 被解冻的账户。
	AccountID string `json:"account_id"`
	// ReferenceID 业务关联 ID。
	ReferenceID string `json:"reference_id"`
	// Amount 解冻金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
	// Reason 解冻原因（CANCELLED/EXPIRED/SETTLED）。
	Reason string `json:"reason"`
}

// FundsSettledEvent 资金结算事件。
// 触发时机：冻结资金转为实际扣款。
type FundsSettledEvent struct {
	LedgerEvent
	// AccountID 结算账户。
	AccountID string `json:"account_id"`
	// ReferenceID 业务关联 ID。
	ReferenceID string `json:"reference_id"`
	// Amount 结算金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
	// JournalID 关联的凭证 ID。
	JournalID string `json:"journal_id"`
}

// AccountCreatedEvent 账户创建事件。
type AccountCreatedEvent struct {
	LedgerEvent
	// AccountID 新账户标识。
	AccountID string `json:"account_id"`
	// AccountType 账户科目类型。
	AccountType AccountType `json:"account_type"`
	// Currency 币种。
	Currency string `json:"currency"`
	// OwnerID 账户所有者 ID。
	OwnerID string `json:"owner_id"`
	// OwnerType 所有者类型（USER/SYSTEM/MERCHANT）。
	OwnerType string `json:"owner_type"`
}

// DayEndReconciledEvent 日终对账完成事件。
// 触发时机：每日 EOD 对账流程完成后发布。
type DayEndReconciledEvent struct {
	LedgerEvent
	// ReconcileDate 对账日期。
	ReconcileDate time.Time `json:"reconcile_date"`
	// TotalAccounts 参与对账的账户数。
	TotalAccounts int `json:"total_accounts"`
	// MatchedCount 匹配成功数。
	MatchedCount int `json:"matched_count"`
	// MismatchCount 不匹配数。
	MismatchCount int `json:"mismatch_count"`
	// TotalDebit 借方合计。
	TotalDebit decimal.Decimal `json:"total_debit"`
	// TotalCredit 贷方合计。
	TotalCredit decimal.Decimal `json:"total_credit"`
	// IsBalanced 是否平衡。
	IsBalanced bool `json:"is_balanced"`
}

// ReversalPostedEvent 冲正事件。
// 触发时机：错误凭证需要冲正时发布。
type ReversalPostedEvent struct {
	LedgerEvent
	// OriginalJournalID 原始凭证 ID。
	OriginalJournalID string `json:"original_journal_id"`
	// ReversalJournalID 冲正凭证 ID。
	ReversalJournalID string `json:"reversal_journal_id"`
	// Reason 冲正原因。
	Reason string `json:"reason"`
	// Amount 冲正金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
}

// InterestAccruedEvent 利息计提事件。
type InterestAccruedEvent struct {
	LedgerEvent
	// AccountID 计息账户。
	AccountID string `json:"account_id"`
	// Principal 本金。
	Principal decimal.Decimal `json:"principal"`
	// Rate 利率（年化）。
	Rate decimal.Decimal `json:"rate"`
	// Interest 计提利息金额。
	Interest decimal.Decimal `json:"interest"`
	// Currency 币种。
	Currency string `json:"currency"`
	// AccrualDate 计息日期。
	AccrualDate time.Time `json:"accrual_date"`
	// JournalID 关联凭证 ID。
	JournalID string `json:"journal_id"`
}

// FeeChargedEvent 手续费扣收事件。
type FeeChargedEvent struct {
	LedgerEvent
	// AccountID 扣费账户。
	AccountID string `json:"account_id"`
	// FeeType 费用类型（TRADING/CUSTODY/MANAGEMENT/WITHDRAWAL）。
	FeeType string `json:"fee_type"`
	// Amount 费用金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
	// ReferenceID 关联业务 ID。
	ReferenceID string `json:"reference_id"`
	// JournalID 关联凭证 ID。
	JournalID string `json:"journal_id"`
}
