package domain

import (
	"github.com/shopspring/decimal"
)

// GridStrategy 网格交易策略
// 包含网格策略的配置参数和运行时状态
type GridStrategy struct {
	StrategyID      string          // 策略 ID
	Symbol          string          // 交易对符号
	UpperPrice      decimal.Decimal // 网格上限价格
	LowerPrice      decimal.Decimal // 网格下限价格
	GridNumber      int             // 网格数量
	QuantityPerGrid decimal.Decimal // 每个网格的交易数量
	Grids           []Grid          // 网格列表
}

// Grid 单个网格
type Grid struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Status   GridStatus // 状态: WAITING(等待中), FILLED(已成交)
	Side     string     // 方向: BUY(买入), SELL(卖出)
}

// GridStatus 网格状态
type GridStatus string

const (
	GridStatusWaiting GridStatus = "WAITING"
	GridStatusFilled  GridStatus = "FILLED"
)

// NewGridStrategy 创建网格策略
func NewGridStrategy(id, symbol string, upper, lower decimal.Decimal, number int, qty decimal.Decimal) *GridStrategy {
	return &GridStrategy{
		StrategyID:      id,
		Symbol:          symbol,
		UpperPrice:      upper,
		LowerPrice:      lower,
		GridNumber:      number,
		QuantityPerGrid: qty,
		Grids:           make([]Grid, 0, number),
	}
}

// InitializeGrids 初始化网格
func (s *GridStrategy) InitializeGrids(currentPrice decimal.Decimal) {
	// 计算网格间距 (等差网格)
	priceRange := s.UpperPrice.Sub(s.LowerPrice)
	interval := priceRange.Div(decimal.NewFromInt(int64(s.GridNumber)))

	for i := 0; i <= s.GridNumber; i++ {
		price := s.LowerPrice.Add(interval.Mul(decimal.NewFromInt(int64(i))))

		// 如果价格低于当前价格，挂买单
		// 如果价格高于当前价格，挂卖单
		var side string
		if price.LessThan(currentPrice) {
			side = "BUY"
		} else {
			side = "SELL"
		}

		s.Grids = append(s.Grids, Grid{
			Price:    price,
			Quantity: s.QuantityPerGrid,
			Status:   GridStatusWaiting,
			Side:     side,
		})
	}
}

// OnPriceUpdate 价格更新处理
// 返回需要执行的订单操作（买入或卖出）
func (s *GridStrategy) OnPriceUpdate(newPrice decimal.Decimal) []GridOrderAction {
	actions := make([]GridOrderAction, 0)

	for i := range s.Grids {
		grid := &s.Grids[i]

		// 简单逻辑：如果价格穿过网格线，且网格处于等待状态，则触发交易
		// 实际生产中需要更复杂的逻辑处理成交确认和网格重置

		// 假设这里只是简单的触发逻辑
		if grid.Status == GridStatusWaiting {
			if grid.Side == "BUY" && newPrice.LessThanOrEqual(grid.Price) {
				// 触发买入
				actions = append(actions, GridOrderAction{
					Side:     "BUY",
					Price:    grid.Price,
					Quantity: grid.Quantity,
				})
				grid.Status = GridStatusFilled
				// 买入成交后，该网格变为等待卖出（通常是在更高一个网格位置，或者原地反向）
				// 这里简化为：买入后，该位置变为等待卖出（如果策略允许同价位反复震荡）
				// 或者更常见的：买入后，在上方网格挂卖单。
				// 为简化，这里仅标记为 Filled
			} else if grid.Side == "SELL" && newPrice.GreaterThanOrEqual(grid.Price) {
				// 触发卖出
				actions = append(actions, GridOrderAction{
					Side:     "SELL",
					Price:    grid.Price,
					Quantity: grid.Quantity,
				})
				grid.Status = GridStatusFilled
			}
		}
	}

	return actions
}

// GridOrderAction 网格订单动作
type GridOrderAction struct {
	Side     string
	Price    decimal.Decimal
	Quantity decimal.Decimal
}
