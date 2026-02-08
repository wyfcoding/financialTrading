// Package domain 算法交易服务仓储接口
package domain

import "context"

type StrategyRepository interface {
	Save(ctx context.Context, strategy *Strategy) error
	GetByID(ctx context.Context, id string) (*Strategy, error)
	ListRunning(ctx context.Context) ([]*Strategy, error) // 获取所有运行中的策略（用于重启恢复）
}

type BacktestRepository interface {
	Save(ctx context.Context, backtest *Backtest) error
	GetByID(ctx context.Context, id string) (*Backtest, error)
}
