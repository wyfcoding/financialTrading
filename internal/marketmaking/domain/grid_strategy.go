package domain

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm"
)

// GridStrategy 网格交易策略
// 包含网格策略配置参数和运行时状态。
// 这里引入了 SkipList（跳表）来优化网格价格的检索效率。
type GridStrategy struct {
	StrategyID      string             // 策略 ID
	Symbol          string             // 交易对符号
	UpperPrice      decimal.Decimal    // 网格上限价格
	LowerPrice      decimal.Decimal    // 网格下限价格
	GridNumber      int                // 网格数量
	QuantityPerGrid decimal.Decimal    // 每个网格的交易数量
	Grids           []Grid             // 网格列表
	PriceIndex      *algorithm.SkipList[float64, int] // 价格索引 (Price -> GridIndex)
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
		PriceIndex:      algorithm.NewSkipList[float64, int](),
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

		grid := Grid{
			Price:    price,
			Quantity: s.QuantityPerGrid,
			Status:   GridStatusWaiting,
			Side:     side,
		}
		s.Grids = append(s.Grids, grid)

		// 将网格价格存入跳表索引，Key 为价格，Value 为网格在切片中的索引
		s.PriceIndex.Insert(price.InexactFloat64(), i)
	}
}

// OnPriceUpdate 价格更新处理
// 返回需要执行的订单操作（买入或卖出）
func (s *GridStrategy) OnPriceUpdate(newPrice decimal.Decimal) []GridOrderAction {
	actions := make([]GridOrderAction, 0)
	priceFloat := newPrice.InexactFloat64()

	// 使用跳表迭代器快速遍历受影响的价格区间，不再需要类型断言。
	it := s.PriceIndex.Iterator()
	for {
		p, gridIdx, ok := it.Next()
		if !ok {
			break
		}

		grid := &s.Grids[gridIdx]

		if grid.Status == GridStatusWaiting {
			// 触发买入：价格下跌穿过网格价格
			if grid.Side == "BUY" && priceFloat <= p {
				actions = append(actions, GridOrderAction{
					Side:     "BUY",
					Price:    grid.Price,
					Quantity: grid.Quantity,
				})
				grid.Status = GridStatusFilled
			} else if grid.Side == "SELL" && priceFloat >= p {
				// 触发卖出：价格上涨穿过网格价格
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
