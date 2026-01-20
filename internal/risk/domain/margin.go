package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// MarginCalculator 保证金计算接口
type MarginCalculator interface {
	CalculateRequiredMargin(ctx context.Context, symbol string, positionValue decimal.Decimal) (decimal.Decimal, error)
}

// VolatilityAdjustedMarginCalculator 基于波动率的动态保证金计算器
type VolatilityAdjustedMarginCalculator struct {
	baseRate             decimal.Decimal
	volatilityMultiplier decimal.Decimal
	marketData           MarketDataProvider
}

func NewVolatilityAdjustedMarginCalculator(baseRate, volMultiplier decimal.Decimal, marketData MarketDataProvider) *VolatilityAdjustedMarginCalculator {
	return &VolatilityAdjustedMarginCalculator{
		baseRate:             baseRate,
		volatilityMultiplier: volMultiplier,
		marketData:           marketData,
	}
}

func (c *VolatilityAdjustedMarginCalculator) CalculateRequiredMargin(ctx context.Context, symbol string, positionValue decimal.Decimal) (decimal.Decimal, error) {
	// 1. 获取当前波动率 (演示逻辑：从 MarketData 获取)
	volatility, err := c.marketData.GetVolatility(ctx, symbol)
	if err != nil {
		// 如果获取失败，降级使用基础利率计算
		return positionValue.Mul(c.baseRate), nil
	}

	// 2. MarginRate = BaseRate + VolMultiplier * Volatility
	marginRate := c.baseRate.Add(c.volatilityMultiplier.Mul(volatility))

	// 限制最高保证金比例 (e.g. 100%)
	if marginRate.GreaterThan(decimal.NewFromInt(1)) {
		marginRate = decimal.NewFromInt(1)
	}

	return positionValue.Mul(marginRate), nil
}

// MarketDataProvider 风险服务所需的行情数据接口
type MarketDataProvider interface {
	GetVolatility(ctx context.Context, symbol string) (decimal.Decimal, error)
}
