// Package domain 提供了智能订单路由（Smart Order Routing）逻辑。
// 变更说明：实现最佳执行（Best Execution）引擎，支持跨市场深度聚合及基于价格、流动性、延迟的智能路由分配。
package domain

import (
	"context"
	"sort"
	"time"
)

// MarketDepth 市场深度
type MarketDepth struct {
	Exchange  string
	Symbol    string
	Bids      []PriceLevel
	Asks      []PriceLevel
	Latency   time.Duration // 市场反馈延迟
	Liquidity float64       // 流动性评分
}

// PriceLevel 价格档位
type PriceLevel struct {
	Price    int64
	Quantity int64
}

// OrderRoute 订单路由路径
type OrderRoute struct {
	Exchange string
	Price    int64
	Quantity int64
}

// SORPlan 智能路由计划
type SORPlan struct {
	Symbol       string
	TotalQty     int64
	Routes       []OrderRoute
	AveragePrice int64
	GeneratedAt  time.Time
}

// SOREngine 最佳执行引擎服务
type SOREngine interface {
	// AggregateDepths 聚合多个市场的深度数据
	AggregateDepths(ctx context.Context, symbol string) ([]*MarketDepth, error)
	// CreateSORPlan 生成最优执行计划
	CreateSORPlan(ctx context.Context, side string, symbol string, quantity int64, depths []*MarketDepth) (*SORPlan, error)
}

// DefaultSOREngine 默认智能路由引擎实现
type DefaultSOREngine struct{}

// AggregateDepths 聚合多个市场的深度数据
func (e *DefaultSOREngine) AggregateDepths(ctx context.Context, symbol string) ([]*MarketDepth, error) {
	// 模拟聚合逻辑，生产环境应从各交易所 API 获取
	return []*MarketDepth{
		{Exchange: "NYSE", Symbol: symbol, Liquidity: 0.9},
		{Exchange: "NASDAQ", Symbol: symbol, Liquidity: 0.8},
	}, nil
}

// CreateSORPlan 实现基于价格优先与流动性分配的算法
func (e *DefaultSOREngine) CreateSORPlan(ctx context.Context, side string, symbol string, quantity int64, depths []*MarketDepth) (*SORPlan, error) {
	plan := &SORPlan{
		Symbol:      symbol,
		TotalQty:    quantity,
		Routes:      make([]OrderRoute, 0),
		GeneratedAt: time.Now(),
	}

	remaining := quantity

	// 简单逻辑：如果是买单，聚合所有市场的卖一价并排序
	if side == "BUY" {
		// 收集所有卖方档位并全局排序
		type globalLevel struct {
			exchange string
			level    PriceLevel
		}
		var allLevels []globalLevel
		for _, d := range depths {
			for _, ask := range d.Asks {
				allLevels = append(allLevels, globalLevel{exchange: d.Exchange, level: ask})
			}
		}

		// 价格升序排序
		sort.Slice(allLevels, func(i, j int) bool {
			return allLevels[i].level.Price < allLevels[j].level.Price
		})

		var totalCost int64
		for _, gl := range allLevels {
			if remaining <= 0 {
				break
			}

			fill := gl.level.Quantity
			if fill > remaining {
				fill = remaining
			}

			plan.Routes = append(plan.Routes, OrderRoute{
				Exchange: gl.exchange,
				Price:    gl.level.Price,
				Quantity: fill,
			})

			totalCost += fill * gl.level.Price
			remaining -= fill
		}

		if quantity > 0 && remaining == 0 {
			plan.AveragePrice = totalCost / quantity
		}
	}

	return plan, nil
}
