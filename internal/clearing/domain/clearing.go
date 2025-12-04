// Package domain 包含清算服务的领域模型、仓储接口和领域服务。
// 这是领域驱动设计（DDD）中的核心层，负责表达业务概念、规则和状态。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 定义清算状态常量，避免在代码中使用魔法字符串。
const (
	SettlementStatusPending   = "PENDING"   // 待处理
	SettlementStatusCompleted = "COMPLETED" // 已完成
	SettlementStatusFailed    = "FAILED"    // 失败
)

// 定义日终清算任务状态常量。
const (
	ClearingStatusProcessing = "PROCESSING" // 处理中
	ClearingStatusCompleted  = "COMPLETED"  // 已完成
	ClearingStatusFailed     = "FAILED"     // 失败
	ClearingStatusPartial    = "PARTIAL"    // 部分完成
)

// Settlement 是清算记录的领域实体（Entity）。
// 它代表一笔交易在清算后的最终状态，是系统中的一个重要事实记录。
type Settlement struct {
	gorm.Model // 嵌入 gorm.Model，包含 ID, CreatedAt, UpdatedAt, DeletedAt 字段

	// SettlementID 是清算的唯一标识符，由系统生成。
	SettlementID string `gorm:"column:settlement_id;type:varchar(50);uniqueIndex;not null" json:"settlement_id"`
	// TradeID 是关联的原始交易ID。
	TradeID string `gorm:"column:trade_id;type:varchar(50);index;not null" json:"trade_id"`
	// BuyUserID 是交易的买方用户ID。
	BuyUserID string `gorm:"column:buy_user_id;type:varchar(50);index;not null" json:"buy_user_id"`
	// SellUserID 是交易的卖方用户ID。
	SellUserID string `gorm:"column:sell_user_id;type:varchar(50);index;not null" json:"sell_user_id"`
	// Symbol 是交易对，例如 "BTC/USDT"。
	Symbol string `gorm:"column:symbol;type:varchar(50);not null" json:"symbol"`
	// Quantity 是成交数量，使用 decimal 类型以保证精度。
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// Price 是成交价格，使用 decimal 类型以保证精度。
	Price decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// Status 是清算状态，使用预定义的常量 (e.g., SettlementStatusCompleted)。
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// SettlementTime 是清算完成的时间。
	SettlementTime time.Time `gorm:"column:settlement_time;type:datetime;not null" json:"settlement_time"`
	// CreatedAt 是记录创建时间，由 gorm.Model 提供。
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
}

// EODClearing 是日终清算（End-of-Day Clearing）的领域实体。
// 它代表一个批处理任务，用于清算一个交易日内的所有交易。
type EODClearing struct {
	gorm.Model // 嵌入 gorm.Model

	// ClearingID 是日终清算任务的唯一标识符。
	ClearingID string `gorm:"column:clearing_id;type:varchar(50);uniqueIndex;not null" json:"clearing_id"`
	// ClearingDate 是清算的日期，格式为 "YYYY-MM-DD"。
	ClearingDate string `gorm:"column:clearing_date;type:varchar(20);index;not null" json:"clearing_date"`
	// Status 是任务的整体状态 (e.g., ClearingStatusProcessing)。
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// StartTime 是任务开始的时间。
	StartTime time.Time `gorm:"column:start_time;type:datetime;not null" json:"start_time"`
	// EndTime 是任务结束的时间，如果任务尚未结束，则为 null。
	EndTime *time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`
	// TradesSettled 是已成功清算的交易数量。
	TradesSettled int64 `gorm:"column:trades_settled;type:bigint;not null" json:"trades_settled"`
	// TotalTrades 是需要清算的总交易数量。
	TotalTrades int64 `gorm:"column:total_trades;type:bigint;not null" json:"total_trades"`
}

// SettlementRepository 是清算记录的仓储接口（Repository Interface）。
// 它定义了对 Settlement 实体的持久化操作，解耦了领域层和基础设施层。
type SettlementRepository interface {
	// Save 保存一个新的或更新一个已有的清算记录。
	Save(ctx context.Context, settlement *Settlement) error
	// Get 根据 SettlementID 获取一个清算记录。
	Get(ctx context.Context, settlementID string) (*Settlement, error)
	// GetByUser 分页获取指定用户的清算历史记录。
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Settlement, int64, error)
	// GetByTrade 根据 TradeID 获取一个清算记录。
	GetByTrade(ctx context.Context, tradeID string) (*Settlement, error)
}

// EODClearingRepository 是日终清算任务的仓储接口。
type EODClearingRepository interface {
	// Save 保存一个新的或更新一个已有的日终清算任务。
	Save(ctx context.Context, clearing *EODClearing) error
	// Get 根据 ClearingID 获取一个日终清算任务。
	Get(ctx context.Context, clearingID string) (*EODClearing, error)
	// GetLatest 获取最新的一次日终清算任务。
	GetLatest(ctx context.Context) (*EODClearing, error)
	// Update 更新一个日终清算任务的状态和进度。
	Update(ctx context.Context, clearing *EODClearing) error
}
