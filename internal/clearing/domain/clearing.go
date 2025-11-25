// Package domain 包含清算服务的领域模型
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Settlement 清算记录实体
type Settlement struct {
	gorm.Model
	// 清算 ID
	SettlementID string `gorm:"column:settlement_id;type:varchar(50);uniqueIndex;not null" json:"settlement_id"`
	// 交易 ID
	TradeID string `gorm:"column:trade_id;type:varchar(50);index;not null" json:"trade_id"`
	// 买方用户 ID
	BuyUserID string `gorm:"column:buy_user_id;type:varchar(50);index;not null" json:"buy_user_id"`
	// 卖方用户 ID
	SellUserID string `gorm:"column:sell_user_id;type:varchar(50);index;not null" json:"sell_user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);not null" json:"symbol"`
	// 成交数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 成交价格
	Price decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// 清算状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// 清算时间
	SettlementTime time.Time `gorm:"column:settlement_time;type:datetime;not null" json:"settlement_time"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
}

// EODClearing 日终清算实体
type EODClearing struct {
	gorm.Model
	// 清算 ID
	ClearingID string `gorm:"column:clearing_id;type:varchar(50);uniqueIndex;not null" json:"clearing_id"`
	// 清算日期
	ClearingDate string `gorm:"column:clearing_date;type:varchar(20);index;not null" json:"clearing_date"`
	// 清算状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// 开始时间
	StartTime time.Time `gorm:"column:start_time;type:datetime;not null" json:"start_time"`
	// 结束时间
	EndTime *time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`
	// 已清算交易数
	TradesSettled int64 `gorm:"column:trades_settled;type:bigint;not null" json:"trades_settled"`
	// 总交易数
	TotalTrades int64 `gorm:"column:total_trades;type:bigint;not null" json:"total_trades"`
}

// SettlementRepository 清算记录仓储接口
type SettlementRepository interface {
	// 保存清算记录
	Save(ctx context.Context, settlement *Settlement) error
	// 获取清算记录
	Get(ctx context.Context, settlementID string) (*Settlement, error)
	// 获取用户清算历史
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Settlement, int64, error)
	// 获取交易清算记录
	GetByTrade(ctx context.Context, tradeID string) (*Settlement, error)
}

// EODClearingRepository 日终清算仓储接口
type EODClearingRepository interface {
	// 保存日终清算
	Save(ctx context.Context, clearing *EODClearing) error
	// 获取日终清算
	Get(ctx context.Context, clearingID string) (*EODClearing, error)
	// 获取最新日终清算
	GetLatest(ctx context.Context) (*EODClearing, error)
	// 更新日终清算
	Update(ctx context.Context, clearing *EODClearing) error
}
