// 变更说明：新增 ledger 读模型仓储接口，支持 CQRS 查询侧。
// 读模型通过 Redis 缓存实时余额，通过 Elasticsearch 支持流水搜索与审计查询。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// LedgerReadRepository 账本读模型仓储接口。
// 用于 CQRS 查询侧，提供高性能的余额查询和流水搜索能力。
type LedgerReadRepository interface {
	// GetCachedBalance 从缓存获取账户余额（O(1) 查询）。
	GetCachedBalance(ctx context.Context, accountID, currency string) (*AccountBalanceView, error)
	// SetCachedBalance 更新缓存中的账户余额。
	SetCachedBalance(ctx context.Context, view *AccountBalanceView) error
	// GetAccountSummary 获取用户所有账户的汇总视图。
	GetAccountSummary(ctx context.Context, ownerID string) ([]*AccountBalanceView, error)
	// GetRecentEntries 获取账户最近的分录（用于首页展示）。
	GetRecentEntries(ctx context.Context, accountID string, limit int) ([]*EntryView, error)
}

// LedgerSearchRepository 账本搜索仓储接口。
// 基于 Elasticsearch 实现，支持复杂的流水搜索和审计查询。
type LedgerSearchRepository interface {
	// IndexJournal 索引凭证到搜索引擎。
	IndexJournal(ctx context.Context, journal *JournalSearchDoc) error
	// IndexEntry 索引分录到搜索引擎。
	IndexEntry(ctx context.Context, entry *EntrySearchDoc) error
	// SearchEntries 搜索分录（支持多维度过滤）。
	SearchEntries(ctx context.Context, query *EntrySearchQuery) (*EntrySearchResult, error)
	// SearchJournals 搜索凭证。
	SearchJournals(ctx context.Context, query *JournalSearchQuery) (*JournalSearchResult, error)
	// GetAuditTrail 获取审计追踪（按时间线展示所有变动）。
	GetAuditTrail(ctx context.Context, accountID string, start, end time.Time) ([]*AuditTrailItem, error)
}

// AccountBalanceView 账户余额读模型视图。
type AccountBalanceView struct {
	// AccountID 账户标识。
	AccountID string `json:"account_id"`
	// AccountName 账户名称。
	AccountName string `json:"account_name"`
	// AccountType 科目类型。
	AccountType string `json:"account_type"`
	// Currency 币种。
	Currency string `json:"currency"`
	// AvailableBalance 可用余额。
	AvailableBalance decimal.Decimal `json:"available_balance"`
	// HoldBalance 冻结余额。
	HoldBalance decimal.Decimal `json:"hold_balance"`
	// TotalBalance 总余额。
	TotalBalance decimal.Decimal `json:"total_balance"`
	// LastUpdated 最后更新时间。
	LastUpdated time.Time `json:"last_updated"`
}

// EntryView 分录读模型视图。
type EntryView struct {
	// EntryID 分录标识。
	EntryID string `json:"entry_id"`
	// JournalID 凭证标识。
	JournalID string `json:"journal_id"`
	// Direction 借贷方向。
	Direction string `json:"direction"`
	// Amount 金额。
	Amount decimal.Decimal `json:"amount"`
	// Currency 币种。
	Currency string `json:"currency"`
	// BalanceAfter 记账后余额。
	BalanceAfter decimal.Decimal `json:"balance_after"`
	// Description 描述。
	Description string `json:"description"`
	// JournalType 凭证类型。
	JournalType string `json:"journal_type"`
	// CreatedAt 创建时间。
	CreatedAt time.Time `json:"created_at"`
}

// JournalSearchDoc 凭证搜索文档。
type JournalSearchDoc struct {
	JournalID     string          `json:"journal_id"`
	TransactionID string          `json:"transaction_id"`
	JournalType   string          `json:"journal_type"`
	Description   string          `json:"description"`
	ReferenceID   string          `json:"reference_id"`
	ReferenceType string          `json:"reference_type"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	Currency      string          `json:"currency"`
	PostedBy      string          `json:"posted_by"`
	IsReversed    bool            `json:"is_reversed"`
	CreatedAt     time.Time       `json:"created_at"`
}

// EntrySearchDoc 分录搜索文档。
type EntrySearchDoc struct {
	EntryID       string          `json:"entry_id"`
	JournalID     string          `json:"journal_id"`
	TransactionID string          `json:"transaction_id"`
	AccountID     string          `json:"account_id"`
	Direction     string          `json:"direction"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	Description   string          `json:"description"`
	JournalType   string          `json:"journal_type"`
	CreatedAt     time.Time       `json:"created_at"`
}

// EntrySearchQuery 分录搜索查询条件。
type EntrySearchQuery struct {
	AccountID     string     `json:"account_id,omitempty"`
	Currency      string     `json:"currency,omitempty"`
	Direction     string     `json:"direction,omitempty"`
	JournalType   string     `json:"journal_type,omitempty"`
	MinAmount     *decimal.Decimal `json:"min_amount,omitempty"`
	MaxAmount     *decimal.Decimal `json:"max_amount,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	Keyword       string     `json:"keyword,omitempty"`
	TransactionID string     `json:"transaction_id,omitempty"`
	Page          int        `json:"page"`
	PageSize      int        `json:"page_size"`
}

// EntrySearchResult 分录搜索结果。
type EntrySearchResult struct {
	Entries  []*EntrySearchDoc `json:"entries"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// JournalSearchQuery 凭证搜索查询条件。
type JournalSearchQuery struct {
	JournalType   string     `json:"journal_type,omitempty"`
	ReferenceID   string     `json:"reference_id,omitempty"`
	ReferenceType string     `json:"reference_type,omitempty"`
	PostedBy      string     `json:"posted_by,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	IsReversed    *bool      `json:"is_reversed,omitempty"`
	Keyword       string     `json:"keyword,omitempty"`
	Page          int        `json:"page"`
	PageSize      int        `json:"page_size"`
}

// JournalSearchResult 凭证搜索结果。
type JournalSearchResult struct {
	Journals []*JournalSearchDoc `json:"journals"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// AuditTrailItem 审计追踪条目。
type AuditTrailItem struct {
	Timestamp     time.Time       `json:"timestamp"`
	EventType     string          `json:"event_type"`
	JournalID     string          `json:"journal_id"`
	TransactionID string          `json:"transaction_id"`
	Direction     string          `json:"direction"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	BalanceBefore decimal.Decimal `json:"balance_before"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	Description   string          `json:"description"`
	PostedBy      string          `json:"posted_by"`
}
