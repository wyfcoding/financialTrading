// Package domain 做市服务的领域模型
package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// StrategyStatus 策略状态
type StrategyStatus string

const (
	StrategyStatusActive StrategyStatus = "ACTIVE" // 激活
	StrategyStatusPaused StrategyStatus = "PAUSED" // 暂停
)

// QuoteStrategy 报价策略实体
type QuoteStrategy struct {
	gorm.Model
	// Symbol 交易对符号
	Symbol string `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null" json:"symbol"`
	// Spread 买卖价差
	Spread decimal.Decimal `gorm:"column:spread;type:decimal(32,18);not null" json:"spread"`
	// MinOrderSize 最小下单量
	MinOrderSize decimal.Decimal `gorm:"column:min_order_size;type:decimal(32,18);not null" json:"min_order_size"`
	// MaxOrderSize 最大下单量
	MaxOrderSize decimal.Decimal `gorm:"column:max_order_size;type:decimal(32,18);not null" json:"max_order_size"`
	// MaxPosition 最大持仓量
	MaxPosition decimal.Decimal `gorm:"column:max_position;type:decimal(32,18);not null" json:"max_position"`
	// Status 策略状态
	Status StrategyStatus `gorm:"column:status;type:varchar(20);default:'ACTIVE'" json:"status"`
}

// MarketMakingPerformance 做市绩效实体
type MarketMakingPerformance struct {
	gorm.Model
	// Symbol 交易对符号
	Symbol string `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null" json:"symbol"`
	// TotalPnL 总盈亏
	TotalPnL decimal.Decimal `gorm:"column:total_pnl;type:decimal(32,18);default:0" json:"total_pnl"`
	// TotalVolume 总成交量
	TotalVolume decimal.Decimal `gorm:"column:total_volume;type:decimal(32,18);default:0" json:"total_volume"`
	// TotalTrades 总成交笔数
	TotalTrades int64 `gorm:"column:total_trades;type:bigint;default:0" json:"total_trades"`
	// SharpeRatio 夏普比率
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(32,18);default:0" json:"sharpe_ratio"`
	// StartTime 开始时间
	StartTime time.Time `gorm:"column:start_time;type:datetime" json:"start_time"`
	// EndTime 结束时间
	EndTime time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`
}

// End of domain file
