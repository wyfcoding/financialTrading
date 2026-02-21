// 变更说明：
// 1. 【增强】多币种（Multi-Currency）复式记账引擎，保证按币种借贷必相等。
// 2. 【增强】引入图表账户体系（Chart of Accounts），支持总账（GL）、明细账（SL）及客户资金隔离（Client Money Segregation）。
// 3. 【增强】支持表内/表外科目，支持冻结资金和账务冲正（Reversal）。
// 4. 【增强】添加日终结算（End-of-Day, EOD）快照接口支持。
package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrInsufficientBalance  = errors.New("insufficient available balance")
	ErrUnbalancedJournal    = errors.New("journal is unbalanced (debit != credit)")
	ErrCurrencyMismatch     = errors.New("journal entries contain mixed currencies without fx leg")
	ErrAccountFrozen        = errors.New("account is frozen")
	ErrHoldNotFound         = errors.New("fund hold not found")
	ErrHoldAlreadyProcessed = errors.New("fund hold is already processed")
)

// AccountType 财务分类
type AccountType string

const (
	Asset     AccountType = "ASSET"     // 资产类（客户资金池、银行存款）
	Liability AccountType = "LIABILITY" // 负债类（客户权益、应付账款）
	Equity    AccountType = "EQUITY"    // 所有者权益
	Revenue   AccountType = "REVENUE"   // 损益-收入（手续费收入）
	Expense   AccountType = "EXPENSE"   // 损益-费用
	OffBalance AccountType = "OFF_BALANCE" // 表外科目（用于未结交收、备付金）
)

// Direction 借贷方向
type Direction string

const (
	Debit  Direction = "DEBIT"  // 借方 (资产/费用增加，负债/权益/收入减少)
	Credit Direction = "CREDIT" // 贷方 (负债/权益/收入增加，资产/费用减少)
)

// JournalType 分录类型
type JournalType string

const (
	JournalDeposit     JournalType = "DEPOSIT"      // 充值
	JournalWithdrawal  JournalType = "WITHDRAWAL"   // 提现
	JournalTrade       JournalType = "TRADE"        // 交易结算
	JournalFee         JournalType = "FEE"          // 手续费扣除
	JournalFxSwap      JournalType = "FX_SWAP"      // 换汇
	JournalReversal    JournalType = "REVERSAL"     // 冲正
	JournalEODClearing JournalType = "EOD_CLEARED"  // 日终清算结转
)

// Account 科目账户 (支持明细账/总账结构)
type Account struct {
	ID           string
	ParentID     string          // 如果是明细账，指向总账(GL) ID
	AccountNo    string          // 财务编码
	Name         string          // 账户名
	Type         AccountType     // 科目大类
	Currency     string          // 基础币种
	Balance      decimal.Decimal // 实时余额 (借贷净值，通常资产借为正，负债贷为正)
	HoldBalance  decimal.Decimal // 冻结余额
	Status       string          // ACTIVE, FROZEN, CLOSED
	Version      int64           // 乐观锁
	LastTxID     string          // 笔数防重
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AvailableBalance 计算可用余额 (实际余额 - 冻结)
func (a *Account) AvailableBalance() decimal.Decimal {
	return a.Balance.Sub(a.HoldBalance)
}

// IsActive 检查是否正常
func (a *Account) IsActive() bool {
	return a.Status == "ACTIVE"
}

// LedgerEntry 分录明细
type LedgerEntry struct {
	ID            string          `json:"id"`
	TransactionID string          `json:"transaction_id"`
	AccountID     string          `json:"account_id"`
	Direction     Direction       `json:"direction"`
	Currency      string          `json:"currency"`
	Amount        decimal.Decimal `json:"amount"`
	BalanceAfter  decimal.Decimal `json:"balance_after"` // 记账后余额
	Narration     string          `json:"narration"`     // 摘要
}

// Journal 凭证主表
type Journal struct {
	ID            string          `json:"id"`
	TransactionID string          `json:"transaction_id"`
	JournalType   JournalType     `json:"journal_type"`
	Entries       []LedgerEntry     `json:"entries"`
	IsReversed    bool            `json:"is_reversed"`
	ReversalOf    string          `json:"reversal_of"` // 被冲正的原始流水
	PostedAt      time.Time       `json:"posted_at"`
	Context       map[string]any  `json:"context"`     // 业务上下文 (OrderNo, TradeID)
}

// Validate 严格的复式记账校验（按币种借贷完全平衡）
func (j *Journal) Validate() error {
	if len(j.Entries) < 2 {
		return errors.New("journal must have at least 2 entries")
	}

	// 按币种统计借贷差额
	balances := make(map[string]decimal.Decimal)
	
	for _, entry := range j.Entries {
		if entry.Amount.IsNegative() || entry.Amount.IsZero() {
			return fmt.Errorf("journal entry amount must be strictly positive: %s", entry.Amount.String())
		}
		
		val := balances[entry.Currency]
		if entry.Direction == Debit {
			val = val.Add(entry.Amount)
		} else {
			val = val.Sub(entry.Amount)
		}
		balances[entry.Currency] = val
	}

	// 所有的币种借贷净值必须为零 
	// (如果有换汇 FX_SWAP 交易，内部必须包含专门的 FX 清算过渡科目以保持单币种平衡)
	for currency, diff := range balances {
		if !diff.IsZero() {
			return fmt.Errorf("%w: mismatch of %s for currency %s", ErrUnbalancedJournal, diff.String(), currency)
		}
	}

	return nil
}

// FundHold 资金冻结凭证
// 用于挂单、提现等需要暂时锁住购买力的场景
type FundHold struct {
	HoldID      string          `json:"hold_id" gorm:"primaryKey"`
	AccountID   string          `json:"account_id"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	Reason      string          `json:"reason"`
	Status      string          `json:"status"` // ACTIVE, RELEASED, CAPTURED
	ExpiresAt   time.Time       `json:"expires_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// LedgerRepository 仓储接口
type LedgerRepository interface {
	// Account Operations
	GetAccount(ctx context.Context, id string) (*Account, error)
	GetAccountForUpdate(ctx context.Context, id string) (*Account, error) // 悲观锁 SELECT FOR UPDATE
	UpdateBalance(ctx context.Context, acc *Account) error

	// Journal Operations
	SaveJournal(ctx context.Context, j *Journal) error
	GetJournalByTransaction(ctx context.Context, txID string) (*Journal, error)
	
	// Fund Holds
	CreateHold(ctx context.Context, hold *FundHold) error
	GetHoldForUpdate(ctx context.Context, holdID string) (*FundHold, error)
	UpdateHold(ctx context.Context, hold *FundHold) error

	// End Of Day (EOD)
	GenerateEODSnapshot(ctx context.Context, date time.Time) error
}
