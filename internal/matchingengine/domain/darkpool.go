// 变更说明：新增暗池撮合引擎，基于中间价 (Midpoint) 撮合，不公开订单簿。
// 假设：暗池最小订单量默认为 100 手，采用参考价格的中间价作为撮合价。
package domain

import (
	"container/list"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/types"
)

// DarkPoolEngine 暗池撮合引擎
type DarkPoolEngine struct {
	Symbol          string
	MinOrderQty     decimal.Decimal // 最小订单量门槛
	BuyQueue        *list.List      // 待成交买单
	SellQueue       *list.List      // 待成交卖单
	mutex           sync.RWMutex
	Logger          *slog.Logger
	ReferenceEngine *DisruptionEngine // 用于获取参考订单簿 BBO
}

func NewDarkPoolEngine(symbol string, minQty decimal.Decimal, ref *DisruptionEngine, logger *slog.Logger) *DarkPoolEngine {
	return &DarkPoolEngine{
		Symbol:          symbol,
		MinOrderQty:     minQty,
		BuyQueue:        list.New(),
		SellQueue:       list.New(),
		ReferenceEngine: ref,
		Logger:          logger,
	}
}

// SubmitOrder 提交订单到暗池
func (e *DarkPoolEngine) SubmitOrder(order *types.Order) (*MatchingResult, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	result := &MatchingResult{
		OrderID:           order.OrderID,
		RemainingQuantity: order.Quantity,
		Status:            "ACCEPTED",
	}

	if order.Quantity.LessThan(e.MinOrderQty) {
		result.Status = "REJECTED_SIZE_TOO_SMALL"
		return result, nil
	}

	// 尝试与对侧队列撮合
	if order.Side == types.SideBuy {
		e.match(order, e.SellQueue, result)
		if result.RemainingQuantity.IsPositive() {
			e.BuyQueue.PushBack(order)
		}
	} else {
		e.match(order, e.BuyQueue, result)
		if result.RemainingQuantity.IsPositive() {
			e.SellQueue.PushBack(order)
		}
	}

	return result, nil
}

// match 执行暗池内部撮合
func (e *DarkPoolEngine) match(incoming *types.Order, targetQueue *list.List, result *MatchingResult) {
	// 获取参考中间价
	midpoint, ok := e.getMidpoint()
	if !ok {
		return
	}

	var nextOrder *list.Element
	for el := targetQueue.Front(); el != nil; el = nextOrder {
		nextOrder = el.Next()
		targetOrder := el.Value.(*types.Order)

		// 暗池价格检查：买单价格必须 >= 中间价，卖单价格必须 <= 中间价
		if incoming.Side == types.SideBuy {
			if incoming.Price.LessThan(midpoint) || targetOrder.Price.GreaterThan(midpoint) {
				continue
			}
		} else {
			if incoming.Price.GreaterThan(midpoint) || targetOrder.Price.LessThan(midpoint) {
				continue
			}
		}

		matchQty := decimal.Min(result.RemainingQuantity, targetOrder.Quantity)
		trade := &types.Trade{
			TradeID:   fmt.Sprintf("DP-%d", time.Now().UnixNano()),
			Symbol:    e.Symbol,
			Price:     midpoint, // 始终在中间价成交
			Quantity:  matchQty,
			Timestamp: time.Now().UnixNano(),
		}

		if incoming.Side == types.SideBuy {
			trade.BuyOrderID = incoming.OrderID
			trade.SellOrderID = targetOrder.OrderID
		} else {
			trade.BuyOrderID = targetOrder.OrderID
			trade.SellOrderID = incoming.OrderID
		}

		result.Trades = append(result.Trades, trade)
		result.RemainingQuantity = result.RemainingQuantity.Sub(matchQty)
		targetOrder.Quantity = targetOrder.Quantity.Sub(matchQty)

		if targetOrder.Quantity.IsZero() {
			targetQueue.Remove(el)
		}

		if result.RemainingQuantity.IsZero() {
			result.Status = "MATCHED"
			break
		}
	}

	if len(result.Trades) > 0 && result.RemainingQuantity.IsPositive() {
		result.Status = "PARTIALLY_MATCHED"
	}
}

// getMidpoint 获取参考订单簿的中间价
func (e *DarkPoolEngine) getMidpoint() (decimal.Decimal, bool) {
	if e.ReferenceEngine == nil {
		return decimal.Zero, false
	}

	snapshot := e.ReferenceEngine.GetOrderBookSnapshot(1)
	if len(snapshot.Bids) == 0 || len(snapshot.Asks) == 0 {
		return decimal.Zero, false
	}

	mid := snapshot.Bids[0].Price.Add(snapshot.Asks[0].Price).Div(decimal.NewFromInt(2))
	return mid, true
}
