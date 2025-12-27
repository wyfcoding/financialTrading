// 包 量化服务的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// StrategyStatus 策略状态
type StrategyStatus string

const (
	StrategyStatusActive   StrategyStatus = "ACTIVE"   // 活跃
	StrategyStatusInactive StrategyStatus = "INACTIVE" // 非活跃
)

// Strategy 策略实体
// 定义量化交易策略
type Strategy struct {
	gorm.Model
	ID          string         `gorm:"column:id;type:varchar(32);primaryKey"`
	Name        string         `gorm:"column:name;type:varchar(100);not null"`
	Description string         `gorm:"column:description;type:text"`
	Script      string         `gorm:"column:script;type:text"`
	Status      StrategyStatus `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
}

// BacktestStatus 回测状态
type BacktestStatus string

const (
	BacktestStatusRunning   BacktestStatus = "RUNNING"
	BacktestStatusCompleted BacktestStatus = "COMPLETED"
	BacktestStatusFailed    BacktestStatus = "FAILED"
)

// BacktestResult 回测结果实体
type BacktestResult struct {
	gorm.Model
	ID          string          `gorm:"column:id;type:varchar(32);primaryKey"`
	StrategyID  string          `gorm:"column:strategy_id;type:varchar(32);index;not null"`
	Symbol      string          `gorm:"column:symbol;type:varchar(32);not null"`
	StartTime   int64           `gorm:"column:start_time;type:bigint"`
	EndTime     int64           `gorm:"column:end_time;type:bigint"`
	TotalReturn decimal.Decimal `gorm:"column:total_return;type:decimal(32,18)"`
	MaxDrawdown decimal.Decimal `gorm:"column:max_drawdown;type:decimal(32,18)"`
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(32,18)"`
	TotalTrades int             `gorm:"column:total_trades;type:int"`
	Status      BacktestStatus  `gorm:"column:status;type:varchar(20);default:'RUNNING'"`
}

// StrategyRepository 策略仓储接口
type StrategyRepository interface {
	Save(ctx context.Context, strategy *Strategy) error
	GetByID(ctx context.Context, id string) (*Strategy, error)
}

// BacktestResultRepository 回测结果仓储接口
type BacktestResultRepository interface {
	Save(ctx context.Context, result *BacktestResult) error
	GetByID(ctx context.Context, id string) (*BacktestResult, error)
}

// MarketDataClient 市场数据客户端接口
type MarketDataClient interface {
	GetHistoricalData(ctx context.Context, symbol string, start, end int64) ([]decimal.Decimal, error)
}
