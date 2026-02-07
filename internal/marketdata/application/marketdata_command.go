package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
)

// MarketDataCommandService 处理所有市场数据写入操作（Commands）。
type MarketDataCommandService struct {
	repo        domain.MarketDataRepository
	logger      *slog.Logger
	broadcaster Broadcaster
	publisher   domain.EventPublisher
	history     *HistoryService
}

// Broadcaster 广播接口
type Broadcaster interface {
	Broadcast(topic string, data any) error
}

// NewMarketDataCommandService 构造函数。
func NewMarketDataCommandService(repo domain.MarketDataRepository, logger *slog.Logger, publisher domain.EventPublisher, history *HistoryService) *MarketDataCommandService {
	return &MarketDataCommandService{
		repo:      repo,
		logger:    logger,
		publisher: publisher,
		history:   history,
	}
}

func (s *MarketDataCommandService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// SaveQuote 保存报价数据
func (s *MarketDataCommandService) SaveQuote(ctx context.Context, cmd SaveQuoteCommand) error {
	quote := domain.NewQuote(cmd.Symbol, cmd.BidPrice, cmd.AskPrice, cmd.BidSize, cmd.AskSize, cmd.LastPrice, cmd.LastSize)
	if cmd.Timestamp > 0 {
		quote.Timestamp = time.UnixMilli(cmd.Timestamp)
	}

	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveQuote(txCtx, quote); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}

		// 发布报价更新事件
		event := domain.QuoteUpdatedEvent{
			Symbol:    quote.Symbol,
			BidPrice:  quote.BidPrice.String(),
			AskPrice:  quote.AskPrice.String(),
			BidSize:   quote.BidSize.String(),
			AskSize:   quote.AskSize.String(),
			LastPrice: quote.LastPrice.String(),
			LastSize:  quote.LastSize.String(),
			Timestamp: quote.Timestamp,
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.QuoteUpdatedEventType, quote.Symbol, event)
	})
}

// SaveKline 保存K线数据
func (s *MarketDataCommandService) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveKline(txCtx, kline); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}

		// 发布K线更新事件
		event := domain.KlineUpdatedEvent{
			Symbol:     kline.Symbol,
			Interval:   kline.Interval,
			OpenPrice:  kline.Open.String(),
			HighPrice:  kline.High.String(),
			LowPrice:   kline.Low.String(),
			ClosePrice: kline.Close.String(),
			Volume:     kline.Volume.String(),
			OpenTime:   kline.OpenTime,
			CloseTime:  kline.CloseTime,
			Timestamp:  time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.KlineUpdatedEventType, kline.Symbol, event)
	})
}

// SaveTrade 保存成交数据
func (s *MarketDataCommandService) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	if trade.Timestamp.IsZero() {
		trade.Timestamp = time.Now()
	}
	if trade.ID == "" {
		trade.ID = fmt.Sprintf("MDTRD-%d", idgen.GenID())
	}
	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveTrade(txCtx, trade); err != nil {
			return err
		}

		if s.history != nil {
			s.history.RecordTrade(trade.Symbol, trade.Price, trade.Timestamp)
		}

		if s.publisher == nil {
			return nil
		}

		// 发布交易执行事件
		event := domain.TradeExecutedEvent{
			Symbol:    trade.Symbol,
			Price:     trade.Price.String(),
			Quantity:  trade.Quantity.String(),
			Side:      trade.Side,
			TradeID:   trade.ID,
			Timestamp: trade.Timestamp,
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.TradeExecutedEventType, trade.ID, event)
	})
}

// SaveOrderBook 保存订单簿
func (s *MarketDataCommandService) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	if orderBook.Timestamp.IsZero() {
		orderBook.Timestamp = time.Now()
	}
	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveOrderBook(txCtx, orderBook); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}

		// 构建订单簿事件数据
		bids := make([][2]string, 0, len(orderBook.Bids))
		for _, bid := range orderBook.Bids {
			bids = append(bids, [2]string{bid.Price.String(), bid.Quantity.String()})
		}

		asks := make([][2]string, 0, len(orderBook.Asks))
		for _, ask := range orderBook.Asks {
			asks = append(asks, [2]string{ask.Price.String(), ask.Quantity.String()})
		}

		// 发布订单簿更新事件
		event := domain.OrderBookUpdatedEvent{
			Symbol:    orderBook.Symbol,
			Bids:      bids,
			Asks:      asks,
			Timestamp: orderBook.Timestamp,
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.OrderBookUpdatedEventType, orderBook.Symbol, event)
	})
}

// HandleTradeExecuted 处理成交事件，更新K线
func (s *MarketDataCommandService) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	symbol := event["symbol"].(string)
	price, _ := decimal.NewFromString(event["price"].(string))
	quantity, _ := decimal.NewFromString(event["quantity"].(string))

	intervals := []string{"1m", "5m", "1h"}
	for _, interval := range intervals {
		if err := s.updateOrCreateKline(ctx, symbol, interval, price, quantity); err != nil {
			s.logger.WarnContext(ctx, "failed to update kline", "symbol", symbol, "interval", interval, "error", err)
		}
	}
	return nil
}

func (s *MarketDataCommandService) updateOrCreateKline(ctx context.Context, symbol, interval string, price, quantity decimal.Decimal) error {
	latest, err := s.repo.GetLatestKline(ctx, symbol, interval)
	if err != nil {
		return err
	}

	now := time.Now()
	if latest == nil || now.After(latest.CloseTime) {
		openTime, closeTime := calculateTimeRange(now, interval)
		newKline := domain.NewKline(symbol, interval, openTime, closeTime, price, price, price, price, quantity)
		return s.SaveKline(ctx, newKline)
	}

	latest.Update(price, quantity)
	return s.SaveKline(ctx, latest)
}

func calculateTimeRange(now time.Time, interval string) (time.Time, time.Time) {
	var duration time.Duration
	switch interval {
	case "1m":
		duration = time.Minute
	case "5m":
		duration = 5 * time.Minute
	case "1h":
		duration = time.Hour
	default:
		d, _ := time.ParseDuration(interval)
		duration = d
	}
	openTime := now.Truncate(duration)
	return openTime, openTime.Add(duration)
}
