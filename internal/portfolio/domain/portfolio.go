//go:build legacy_portfolio_simple
// +build legacy_portfolio_simple

package domain

import (
	"context"
	"github.com/shopspring/decimal"
)

type Portfolio struct {
	ID           string
	UserID       string
	BaseCurrency string
	TotalValue   decimal.Decimal
}

type Position struct {
	Symbol   string
	Quantity decimal.Decimal
	AvgCost  decimal.Decimal
}

type PortfolioRepository interface {
	GetByUserID(ctx context.Context, userID string) (*Portfolio, error)
	Save(ctx context.Context, p *Portfolio) error
}

// RebalanceEngine 投资组合再平衡引擎
type RebalanceEngine struct{}

func (e *RebalanceEngine) Calculate(current []Position, targetWeights map[string]decimal.Decimal) interface{} {
	return nil // 高级矩阵算法模型
}
