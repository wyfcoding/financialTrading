package domain

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/types"
)

// MarginService 负责在撮合前/撮合中进行风险检查。
type MarginService interface {
	CheckMargin(ctx context.Context, userID string, order *types.Order) (bool, error)
	GetMarkPrice(ctx context.Context, symbol string) (decimal.Decimal, error)
}

// MockMarginService 模拟实现。
type MockMarginService struct {
	Balances map[string]decimal.Decimal
}

func NewMockMarginService() *MockMarginService {
	return &MockMarginService{
		Balances: make(map[string]decimal.Decimal),
	}
}

func (s *MockMarginService) CheckMargin(ctx context.Context, userID string, order *types.Order) (bool, error) {
	// 简单的全量保证金检查：Price * Quantity / Leverage
	required := order.Price.Mul(order.Quantity)
	if !order.Leverage.IsZero() {
		required = required.Div(order.Leverage)
	}

	balance, ok := s.Balances[userID]
	if !ok {
		// 默认给 100w 模拟资金
		balance = decimal.NewFromInt(1000000)
		s.Balances[userID] = balance
	}

	if balance.LessThan(required) {
		return false, fmt.Errorf("insufficient margin: balance %v, required %v", balance, required)
	}
	return true, nil
}

func (s *MockMarginService) GetMarkPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return decimal.NewFromFloat(100.0), nil
}
