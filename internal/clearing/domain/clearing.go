// 包 domain 清算服务的领域模型、仓储接口和领域服务。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 定义清算状态常量
const (
	SettlementStatusPending   = "PENDING"   // 待处理
	SettlementStatusCompleted = "COMPLETED" // 已完成
	SettlementStatusFailed    = "FAILED"    // 失败
)

// 定义日终清算任务状态常量
const (
	ClearingStatusProcessing = "PROCESSING" // 处理中
	ClearingStatusCompleted  = "COMPLETED"  // 已完成
	ClearingStatusFailed     = "FAILED"     // 失败
	ClearingStatusPartial    = "PARTIAL"    // 部分完成
)

// Settlement 是清算记录的领域实体
type Settlement struct {
	gorm.Model
	// SettlementID 是清算的唯一标识符
	SettlementID string `gorm:"column:settlement_id;type:varchar(32);uniqueIndex;not null" json:"settlement_id"`
	// TradeID 是关联的原始交易ID
	TradeID string `gorm:"column:trade_id;type:varchar(32);index;not null" json:"trade_id"`
	// BuyUserID 是交易的买方用户ID
	BuyUserID string `gorm:"column:buy_user_id;type:varchar(32);index;not null" json:"buy_user_id"`
	// SellUserID 是交易的卖方用户ID
	SellUserID string `gorm:"column:sell_user_id;type:varchar(32);index;not null" json:"sell_user_id"`
	// Symbol 是交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	// Quantity 是成交数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(32,18);not null" json:"quantity"`
	// Price 是成交价格
	Price decimal.Decimal `gorm:"column:price;type:decimal(32,18);not null" json:"price"`
	// Status 是清算状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// SettlementTime 是清算完成的时间
	SettlementTime time.Time `gorm:"column:settlement_time;type:datetime;not null" json:"settlement_time"`
}

// EODClearing 是日终清算的领域实体
type EODClearing struct {
	gorm.Model
	// ClearingID 是日终清算任务的唯一标识符
	ClearingID string `gorm:"column:clearing_id;type:varchar(32);uniqueIndex;not null" json:"clearing_id"`
	// ClearingDate 是清算的日期，格式为 "YYYY-MM-DD"
	ClearingDate string `gorm:"column:clearing_date;type:varchar(20);index;not null" json:"clearing_date"`
	// Status 是任务的整体状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// StartTime 是任务开始的时间
	StartTime time.Time `gorm:"column:start_time;type:datetime;not null" json:"start_time"`
	// EndTime 是任务结束的时间
	EndTime *time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`
	// TradesSettled 是已成功清算的交易数量
	TradesSettled int64 `gorm:"column:trades_settled;type:bigint;not null" json:"trades_settled"`
	// TotalTrades 是需要清算的总交易数量
	TotalTrades int64 `gorm:"column:total_trades;type:bigint;not null" json:"total_trades"`
}

// SettlementRepository 是清算记录的仓储接口
type SettlementRepository interface {
	// Save 保存或更新清算记录
	Save(ctx context.Context, settlement *Settlement) error
	// Get 根据 SettlementID 获取一个清算记录
	Get(ctx context.Context, settlementID string) (*Settlement, error)
	// GetByUser 分页获取指定用户的清算历史记录
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Settlement, int64, error)
	// GetByTrade 根据 TradeID 获取一个清算记录
	GetByTrade(ctx context.Context, tradeID string) (*Settlement, error)
}

// EODClearingRepository 是日终清算任务的仓储接口
type EODClearingRepository interface {
	// Save 保存或更新日终清算任务
	Save(ctx context.Context, clearing *EODClearing) error
	// Get 根据 ClearingID 获取一个日终清算任务
	Get(ctx context.Context, clearingID string) (*EODClearing, error)
	// GetLatest 获取最新的一次日终清算任务
	GetLatest(ctx context.Context) (*EODClearing, error)
	// Update 显式更新日终清算任务信息
	Update(ctx context.Context, clearing *EODClearing) error
}
