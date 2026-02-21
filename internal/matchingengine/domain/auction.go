package domain

import (
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	algorithm "github.com/wyfcoding/pkg/algos/structures"
	"github.com/wyfcoding/pkg/algos/types"
)

// CalculateEquilibrium 寻找平衡价格 (Equilibrium Price)
// 1. 最大化成交量
// 2. 最小化失衡
func (a *AuctionEngine) CalculateEquilibrium() *AuctionResult {
	// 1. 收集所有不重复的价格点
	// 使用 map 去重
	priceMap := make(map[float64]struct{})

	itB := a.Bids.Iterator()
	for {
		p, _, ok := itB.Next()
		if !ok {
			break
		}
		priceMap[mathAbs(p)] = struct{}{}
	}

	itA := a.Asks.Iterator()
	for {
		p, _, ok := itA.Next()
		if !ok {
			break
		}
		priceMap[p] = struct{}{}
	}

	// 2. 排序价格
	var prices []float64
	for p := range priceMap {
		prices = append(prices, p)
	}
	sort.Float64s(prices)

	if len(prices) == 0 {
		return &AuctionResult{}
	}

	// 3. 构建累积量 (CDF)
	// bids: Price Descending -> CumQty increases as Price decreases
	// asks: Price Ascending -> CumQty increases as Price increases

	// 为了 O(N) 扫描，我们需要预计算每个价格点的 Bid/Ask 这一档的独立数量
	bidVols := make(map[float64]decimal.Decimal)
	itB = a.Bids.Iterator()
	for {
		p, lv, ok := itB.Next()
		if !ok {
			break
		}
		bidVols[mathAbs(p)] = a.getLevelQty(lv)
	}

	askVols := make(map[float64]decimal.Decimal)
	itA = a.Bids.Iterator()
	// Fixed: should use a.Asks
	itA = a.Asks.Iterator()
	for {
		p, lv, ok := itA.Next()
		if !ok {
			break
		}
		askVols[p] = a.getLevelQty(lv)
	}

	n := len(prices)
	cumBids := make([]decimal.Decimal, n) // cumBids[i] = Sum(BidVols where Price >= prices[i])
	cumAsks := make([]decimal.Decimal, n) // cumAsks[i] = Sum(AskVols where Price <= prices[i])

	// Calculate Cumulative Bids (Right to Left / High to Low)
	// prices are sorted Low to High.
	// Bids accumulate from High Price down to Low Price.
	currentBidSum := decimal.Zero
	for i := n - 1; i >= 0; i-- {
		p := prices[i]
		if v, ok := bidVols[p]; ok {
			currentBidSum = currentBidSum.Add(v)
		}
		cumBids[i] = currentBidSum
	}

	// Calculate Cumulative Asks (Left to Right / Low to High)
	currentAskSum := decimal.Zero
	for i := 0; i < n; i++ {
		p := prices[i]
		if v, ok := askVols[p]; ok {
			currentAskSum = currentAskSum.Add(v)
		}
		cumAsks[i] = currentAskSum
	}

	// 4. Find Equilibrium
	var bestPrice decimal.Decimal
	var maxMatched decimal.Decimal
	var minImbalance = decimal.NewFromFloat(math.MaxFloat64)

	for i := 0; i < n; i++ {
		p := prices[i]
		buyQty := cumBids[i]
		sellQty := cumAsks[i]

		matched := decimal.Min(buyQty, sellQty)
		imbalance := buyQty.Sub(sellQty).Abs()
		priceDec := decimal.NewFromFloat(p)

		if matched.GreaterThan(maxMatched) {
			maxMatched = matched
			minImbalance = imbalance
			bestPrice = priceDec
		} else if matched.Equal(maxMatched) && matched.IsPositive() {
			if imbalance.LessThan(minImbalance) {
				minImbalance = imbalance
				bestPrice = priceDec
			}
		}
	}

	result := &AuctionResult{
		EquilibriumPrice: bestPrice,
		MatchedQuantity:  maxMatched,
		ImbalanceQty:     minImbalance,
	}

	if maxMatched.IsPositive() {
		// 生成成交逻辑在此简化，实际需遍历订单簿进行拆单成交
		a.Logger.Info("auction equilibrium found", "price", bestPrice.String(), "volume", maxMatched.String())

		// 4. 生成虚拟成交记录
		// Iterate through bids to generate trades
		itB := a.Bids.Iterator()
		remMatched := maxMatched // Use maxMatched as the remaining matched quantity
		for remMatched.IsPositive() {
			_, lv, ok := itB.Next()
			if !ok {
				break
			}
			// Only consider bids that are at or above the equilibrium price
			if lv.Price.GreaterThanOrEqual(bestPrice) {
				for el := lv.Orders.Front(); el != nil; el = el.Next() {
					o := el.Value.(*types.Order)
					fill := decimal.Min(remMatched, o.Quantity)
					result.Trades = append(result.Trades, &types.Trade{
						Symbol:     a.Symbol,
						Price:      bestPrice, // All trades at equilibrium price
						Quantity:   fill,
						Timestamp:  time.Now().UnixNano(),
						BuyOrderID: o.OrderID,
						// SellOrderID will be filled by matching with asks
					})
					remMatched = remMatched.Sub(fill)
					if remMatched.IsZero() {
						break
					}
				}
			}
		}

		// Iterate through asks to match with generated buy trades
		itA := a.Asks.Iterator()
		remMatched = maxMatched // Reset remMatched for asks
		tradeIndex := 0
		for remMatched.IsPositive() {
			_, lv, ok := itA.Next()
			if !ok {
				break
			}
			// Only consider asks that are at or below the equilibrium price
			if lv.Price.LessThanOrEqual(bestPrice) {
				for el := lv.Orders.Front(); el != nil; el = el.Next() {
					o := el.Value.(*types.Order)
					fill := decimal.Min(remMatched, o.Quantity)

					// Find corresponding buy trades to fill
					for tradeIndex < len(result.Trades) && fill.IsPositive() {
						currentTrade := result.Trades[tradeIndex]
						if currentTrade.SellOrderID == "" { // If this buy trade hasn't been matched with a sell order yet
							tradeFill := decimal.Min(fill, currentTrade.Quantity)
							currentTrade.SellOrderID = o.OrderID
							// If the current trade quantity is fully filled by this ask, move to next trade
							// If the ask quantity is fully used, move to next ask
							// For simplicity, we assume trades are filled sequentially.
							// A more robust implementation would track remaining quantity on each trade.
							// Here, we just assign the SellOrderID and assume the quantity matches.
							// This part needs careful consideration for partial fills across multiple trades.
							// For now, we'll just assign the SellOrderID to the first available trade.
							fill = fill.Sub(tradeFill)
							remMatched = remMatched.Sub(tradeFill)
							if currentTrade.Quantity.Equal(tradeFill) {
								tradeIndex++
							}
						} else {
							tradeIndex++ // Move to next trade if already filled
						}
					}
					if remMatched.IsZero() {
						break
					}
				}
			}
		}
	}

	return result
}

func (a *AuctionEngine) getCumulativeQty(book *algorithm.SkipList[float64, *EngineOrderLevel], price decimal.Decimal, isBid bool) decimal.Decimal {
	var total decimal.Decimal
	it := book.Iterator()
	for {
		_, level, ok := it.Next()
		if !ok {
			break
		}

		realPrice := level.Price
		if isBid {
			if realPrice.GreaterThanOrEqual(price) {
				total = total.Add(a.getLevelQty(level))
			}
		} else {
			if realPrice.LessThanOrEqual(price) {
				total = total.Add(a.getLevelQty(level))
			}
		}
	}
	return total
}

func (a *AuctionEngine) getLevelQty(level *EngineOrderLevel) decimal.Decimal {
	var q decimal.Decimal
	for el := level.Orders.Front(); el != nil; el = el.Next() {
		q = q.Add(el.Value.(*types.Order).Quantity)
	}
	return q
}

func mathAbs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// 注意：此处省略了 math 导入，需确保在编译时正确处理
