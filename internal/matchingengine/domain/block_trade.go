// 变更说明：新增大宗交易录入与匹配逻辑，支持场外议价后的系统确认与登记。
// 假设：大宗交易由买卖双方通过特定的成交确认号 (Affirmation ID) 进行关联，必须满足最低成交额要求。
package domain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/algorithm/types"
)

// BlockTradeEngine 大宗交易处理引擎
type BlockTradeEngine struct {
	Symbol              string
	MinBlockValue       decimal.Decimal // 大宗交易最低成交额门槛
	PendingNegotiations map[string]*BlockTradeNegotiation
	mutex               sync.Mutex
}

// BlockTradeNegotiation 大宗交易议价单
type BlockTradeNegotiation struct {
	NegotiationID string
	Symbol        string
	Side          types.Side
	Price         decimal.Decimal
	Quantity      decimal.Decimal
	Counterparty  string // 交易对手 ID
	UserID        string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Status        string // PENDING, AFFIRMED, CANCELLED
}

func NewBlockTradeEngine(symbol string, minVal decimal.Decimal) *BlockTradeEngine {
	return &BlockTradeEngine{
		Symbol:              symbol,
		MinBlockValue:       minVal,
		PendingNegotiations: make(map[string]*BlockTradeNegotiation),
	}
}

// ProposeNegotiation 发起大宗交易议价
func (e *BlockTradeEngine) ProposeNegotiation(order *types.Order, counterpartyID string) (*BlockTradeNegotiation, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	tradeValue := order.Price.Mul(order.Quantity)
	if tradeValue.LessThan(e.MinBlockValue) {
		return nil, fmt.Errorf("trade value %s below block threshold %s", tradeValue, e.MinBlockValue)
	}

	negID := fmt.Sprintf("NEG-%d", time.Now().UnixNano())
	neg := &BlockTradeNegotiation{
		NegotiationID: negID,
		Symbol:        order.Symbol,
		Side:          order.Side,
		Price:         order.Price,
		Quantity:      order.Quantity,
		Counterparty:  counterpartyID,
		UserID:        order.UserID,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		Status:        "PENDING",
	}

	e.PendingNegotiations[negID] = neg
	return neg, nil
}

// AffirmNegotiation 确认大宗交易议价（对手方调用）
func (e *BlockTradeEngine) AffirmNegotiation(negID string, userID string) (*types.Trade, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	neg, ok := e.PendingNegotiations[negID]
	if !ok {
		return nil, errors.New("negotiation not found")
	}

	if neg.Counterparty != userID {
		return nil, errors.New("unauthorized counterparty")
	}

	if time.Now().After(neg.ExpiresAt) {
		neg.Status = "EXPIRED"
		return nil, errors.New("negotiation expired")
	}

	neg.Status = "AFFIRMED"

	// 生成成交记录
	trade := &types.Trade{
		TradeID:   fmt.Sprintf("BT-%d", time.Now().UnixNano()),
		Symbol:    neg.Symbol,
		Price:     neg.Price,
		Quantity:  neg.Quantity,
		Timestamp: time.Now().UnixNano(),
	}

	if neg.Side == types.SideBuy {
		trade.BuyOrderID = "BLOCK-" + neg.UserID
		trade.SellOrderID = "BLOCK-" + userID
		trade.BuyUserID = neg.UserID
		trade.SellUserID = userID
	} else {
		trade.BuyOrderID = "BLOCK-" + userID
		trade.SellOrderID = "BLOCK-" + neg.UserID
		trade.BuyUserID = userID
		trade.SellUserID = neg.UserID
	}

	delete(e.PendingNegotiations, negID)
	return trade, nil
}
