package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataCommandService 处理所有市场数据写入操作（Commands）。
type MarketDataCommandService struct {
	repo        domain.MarketDataRepository
	logger      *slog.Logger
	broadcaster Broadcaster
	publisher   domain.EventPublisher
}

// Broadcaster 广播接口
type Broadcaster interface {
	Broadcast(topic string, data any) error
}

// NewMarketDataCommandService 构造函数。
func NewMarketDataCommandService(repo domain.MarketDataRepository, logger *slog.Logger, publisher domain.EventPublisher) *MarketDataCommandService {
	return &MarketDataCommandService{
		repo:      repo,
		logger:    logger,
		publisher: publisher,
	}
}

func (s *MarketDataCommandService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// SaveQuote 保存报价数据
func (s *MarketDataCommandService) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	quote := domain.NewQuote(symbol, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize)
	if err := s.repo.SaveQuote(ctx, quote); err != nil {
		return err
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
	s.publisher.Publish(ctx, "marketdata.quote.updated", symbol, event)

	return nil
}

// SaveKline 保存K线数据
func (s *MarketDataCommandService) SaveKline(ctx context.Context, kline *domain.Kline) error {
	if err := s.repo.SaveKline(ctx, kline); err != nil {
		return err
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
	s.publisher.Publish(ctx, "marketdata.kline.updated", kline.Symbol, event)

	return nil
}

// SaveTrade 保存成交数据
func (s *MarketDataCommandService) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	if err := s.repo.SaveTrade(ctx, trade); err != nil {
		return err
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
	s.publisher.Publish(ctx, "marketdata.trade.executed", trade.Symbol, event)

	return nil
}

// SaveOrderBook 保存订单簿
func (s *MarketDataCommandService) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	if err := s.repo.SaveOrderBook(ctx, orderBook); err != nil {
		return err
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
	s.publisher.Publish(ctx, "marketdata.orderbook.updated", orderBook.Symbol, event)

	return nil
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
		return s.repo.SaveKline(ctx, newKline)
	}

	latest.Update(price, quantity)
	return s.repo.SaveKline(ctx, latest)
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
