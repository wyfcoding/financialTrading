// Package domain 包含做市服务的领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// StrategyStatus 策略状态
type StrategyStatus string

const (
	StrategyStatusActive StrategyStatus = "ACTIVE" // 激活
	StrategyStatusPaused StrategyStatus = "PAUSED" // 暂停
)

// QuoteStrategy 报价策略实体
// 定义了针对特定交易对的做市参数
type QuoteStrategy struct {
	gorm.Model
	ID           string         `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	Symbol       string         `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null" json:"symbol"`
	Spread       float64        `gorm:"column:spread;type:decimal(10,4);not null" json:"spread"`
	MinOrderSize float64        `gorm:"column:min_order_size;type:decimal(20,8);not null" json:"min_order_size"`
	MaxOrderSize float64        `gorm:"column:max_order_size;type:decimal(20,8);not null" json:"max_order_size"`
	MaxPosition  float64        `gorm:"column:max_position;type:decimal(20,8);not null" json:"max_position"`
	Status       StrategyStatus `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

// MarketMakingPerformance 做市绩效实体
type MarketMakingPerformance struct {
	gorm.Model
	Symbol      string    `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null" json:"symbol"`
	TotalPnL    float64   `gorm:"column:total_pnl;type:decimal(20,8);default:0" json:"total_pnl"`
	TotalVolume float64   `gorm:"column:total_volume;type:decimal(20,8);default:0" json:"total_volume"`
	TotalTrades int       `gorm:"column:total_trades;type:int;default:0" json:"total_trades"`
	SharpeRatio float64   `gorm:"column:sharpe_ratio;type:decimal(10,4);default:0" json:"sharpe_ratio"`
	StartTime   time.Time `gorm:"column:start_time;type:datetime" json:"start_time"`
	EndTime     time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`
}

// QuoteStrategyRepository 策略仓储接口
type QuoteStrategyRepository interface {
	Save(ctx context.Context, strategy *QuoteStrategy) error
	GetBySymbol(ctx context.Context, symbol string) (*QuoteStrategy, error)
}

// PerformanceRepository 绩效仓储接口
type PerformanceRepository interface {
	Save(ctx context.Context, performance *MarketMakingPerformance) error
	GetBySymbol(ctx context.Context, symbol string) (*MarketMakingPerformance, error)
}

// OrderClient 订单服务客户端接口
type OrderClient interface {
	PlaceOrder(ctx context.Context, symbol string, side string, price, quantity float64) (string, error)
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetPrice(ctx context.Context, symbol string) (float64, error)
}
