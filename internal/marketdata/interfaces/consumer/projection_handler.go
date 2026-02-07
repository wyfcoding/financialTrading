package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type MarketDataProjectionHandler struct {
	projector *application.MarketDataProjectionService
	logger    *slog.Logger
}

func NewMarketDataProjectionHandler(projector *application.MarketDataProjectionService, logger *slog.Logger) *MarketDataProjectionHandler {
	return &MarketDataProjectionHandler{projector: projector, logger: logger}
}

func (h *MarketDataProjectionHandler) Handle(ctx context.Context, msg kafka.Message) error {
	switch msg.Topic {
	case domain.QuoteUpdatedEventType:
		var event domain.QuoteUpdatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal quote event", "error", err)
			return err
		}
		quote := &domain.Quote{
			Symbol:    event.Symbol,
			BidPrice:  mustDecimal(event.BidPrice),
			AskPrice:  mustDecimal(event.AskPrice),
			BidSize:   mustDecimal(event.BidSize),
			AskSize:   mustDecimal(event.AskSize),
			LastPrice: mustDecimal(event.LastPrice),
			LastSize:  mustDecimal(event.LastSize),
			Timestamp: event.Timestamp,
		}
		return h.projector.ProjectQuote(ctx, quote)
	case domain.KlineUpdatedEventType:
		var event domain.KlineUpdatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal kline event", "error", err)
			return err
		}
		kline := &domain.Kline{
			Symbol:    event.Symbol,
			Interval:  event.Interval,
			OpenTime:  event.OpenTime,
			CloseTime: event.CloseTime,
			Open:      mustDecimal(event.OpenPrice),
			High:      mustDecimal(event.HighPrice),
			Low:       mustDecimal(event.LowPrice),
			Close:     mustDecimal(event.ClosePrice),
			Volume:    mustDecimal(event.Volume),
		}
		return h.projector.ProjectKline(ctx, kline)
	case domain.TradeExecutedEventType:
		var event domain.TradeExecutedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal trade event", "error", err)
			return err
		}
		trade := &domain.Trade{
			ID:        event.TradeID,
			Symbol:    event.Symbol,
			Price:     mustDecimal(event.Price),
			Quantity:  mustDecimal(event.Quantity),
			Side:      event.Side,
			Timestamp: event.Timestamp,
		}
		return h.projector.ProjectTrade(ctx, trade)
	case domain.OrderBookUpdatedEventType:
		var event domain.OrderBookUpdatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.ErrorContext(ctx, "failed to unmarshal orderbook event", "error", err)
			return err
		}
		bids := make([]domain.OrderBookItem, 0, len(event.Bids))
		for _, bid := range event.Bids {
			if len(bid) < 2 {
				continue
			}
			bids = append(bids, domain.OrderBookItem{
				Price:    mustDecimal(bid[0]),
				Quantity: mustDecimal(bid[1]),
			})
		}
		asks := make([]domain.OrderBookItem, 0, len(event.Asks))
		for _, ask := range event.Asks {
			if len(ask) < 2 {
				continue
			}
			asks = append(asks, domain.OrderBookItem{
				Price:    mustDecimal(ask[0]),
				Quantity: mustDecimal(ask[1]),
			})
		}
		ob := &domain.OrderBook{
			Symbol:    event.Symbol,
			Bids:      bids,
			Asks:      asks,
			Timestamp: event.Timestamp,
		}
		return h.projector.ProjectOrderBook(ctx, ob)
	default:
		h.logger.WarnContext(ctx, "unknown marketdata event topic", "topic", msg.Topic)
		return nil
	}
}

func mustDecimal(v string) decimal.Decimal {
	d, _ := decimal.NewFromString(v)
	return d
}
