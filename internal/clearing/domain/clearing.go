// 包 domain 清算服务的领域模型、仓储接口和领域服务。
package domain

import (
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

// MarginRequirement 是保证金要求的领域实体
type MarginRequirement struct {
	gorm.Model
	// Symbol 是交易对代码
	Symbol string `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null" json:"symbol"`
	// BaseMarginRate 是基础保证金率 (例如 0.10 表示 10%)
	BaseMarginRate decimal.Decimal `gorm:"column:base_margin_rate;type:decimal(10,4);not null" json:"base_margin_rate"`
	// VolatilityFactor 是波动率调整因子 (例如 0.05 表示在基准上增加 5%)
	VolatilityFactor decimal.Decimal `gorm:"column:volatility_factor;type:decimal(10,4);not null" json:"volatility_factor"`
	// UpdatedBy 是最后一次更新的操作人或系统
	UpdatedBy string `gorm:"column:updated_by;type:varchar(32)" json:"updated_by"`
}

// CurrentMarginRate 计算当前生效的保证金率
func (m *MarginRequirement) CurrentMarginRate() decimal.Decimal {
	// 公式：Base * (1 + VolatilityFactor)
	return m.BaseMarginRate.Mul(decimal.NewFromFloat(1).Add(m.VolatilityFactor))
}

// End of domain file
