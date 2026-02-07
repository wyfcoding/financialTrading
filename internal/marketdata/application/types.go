package application

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// SaveQuoteCommand 保存报价命令
// source 保留用于追踪来源（当前不落库）
type SaveQuoteCommand struct {
	Symbol    string
	BidPrice  decimal.Decimal
	AskPrice  decimal.Decimal
	BidSize   decimal.Decimal
	AskSize   decimal.Decimal
	LastPrice decimal.Decimal
	LastSize  decimal.Decimal
	Timestamp int64
	Source    string
}

// QuoteDTO 行情数据 DTO
// timestamp 使用毫秒
type QuoteDTO struct {
	Symbol    string `json:"symbol"`
	BidPrice  string `json:"bid_price"`
	AskPrice  string `json:"ask_price"`
	BidSize   string `json:"bid_size"`
	AskSize   string `json:"ask_size"`
	LastPrice string `json:"last_price"`
	LastSize  string `json:"last_size"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
}

// KlineDTO K线数据 DTO
type KlineDTO struct {
	OpenTime  int64  `json:"open_time"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	CloseTime int64  `json:"close_time"`
}

// TradeDTO 成交数据 DTO
type TradeDTO struct {
	TradeID   string `json:"trade_id"`
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

// OrderBookLevelDTO 订单簿档位 DTO
type OrderBookLevelDTO struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// OrderBookDTO 订单簿 DTO
type OrderBookDTO struct {
	Symbol    string              `json:"symbol"`
	Bids      []OrderBookLevelDTO `json:"bids"`
	Asks      []OrderBookLevelDTO `json:"asks"`
	Timestamp int64               `json:"timestamp"`
}

func toQuoteDTO(q *domain.Quote) *QuoteDTO {
	if q == nil {
		return nil
	}
	return &QuoteDTO{
		Symbol:    q.Symbol,
		BidPrice:  q.BidPrice.String(),
		AskPrice:  q.AskPrice.String(),
		BidSize:   q.BidSize.String(),
		AskSize:   q.AskSize.String(),
		LastPrice: q.LastPrice.String(),
		LastSize:  q.LastSize.String(),
		Timestamp: q.Timestamp.UnixMilli(),
	}
}

func toQuoteDTOs(quotes []*domain.Quote) []*QuoteDTO {
	dtos := make([]*QuoteDTO, len(quotes))
	for i, q := range quotes {
		dtos[i] = toQuoteDTO(q)
	}
	return dtos
}

func toKlineDTOs(klines []*domain.Kline) []*KlineDTO {
	dtos := make([]*KlineDTO, len(klines))
	for i, k := range klines {
		dtos[i] = &KlineDTO{
			OpenTime:  k.OpenTime.UnixMilli(),
			Open:      k.Open.String(),
			High:      k.High.String(),
			Low:       k.Low.String(),
			Close:     k.Close.String(),
			Volume:    k.Volume.String(),
			CloseTime: k.CloseTime.UnixMilli(),
		}
	}
	return dtos
}

func toTradeDTOs(trades []*domain.Trade) []*TradeDTO {
	dtos := make([]*TradeDTO, len(trades))
	for i, t := range trades {
		dtos[i] = &TradeDTO{
			TradeID:   t.ID,
			Symbol:    t.Symbol,
			Price:     t.Price.String(),
			Quantity:  t.Quantity.String(),
			Side:      t.Side,
			Timestamp: t.Timestamp.UnixMilli(),
		}
	}
	return dtos
}

func toOrderBookDTO(ob *domain.OrderBook) *OrderBookDTO {
	if ob == nil {
		return nil
	}
	bids := make([]OrderBookLevelDTO, 0, len(ob.Bids))
	for _, bid := range ob.Bids {
		bids = append(bids, OrderBookLevelDTO{
			Price:    bid.Price.String(),
			Quantity: bid.Quantity.String(),
		})
	}
	asks := make([]OrderBookLevelDTO, 0, len(ob.Asks))
	for _, ask := range ob.Asks {
		asks = append(asks, OrderBookLevelDTO{
			Price:    ask.Price.String(),
			Quantity: ask.Quantity.String(),
		})
	}
	return &OrderBookDTO{
		Symbol:    ob.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: ob.Timestamp.UnixMilli(),
	}
}
