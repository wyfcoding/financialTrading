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
}

// Broadcaster 广播接口
type Broadcaster interface {
	Broadcast(topic string, data any) error
}

// NewMarketDataCommandService 构造函数。
func NewMarketDataCommandService(repo domain.MarketDataRepository, logger *slog.Logger) *MarketDataCommandService {
	return &MarketDataCommandService{
		repo:   repo,
		logger: logger,
	}
}

func (s *MarketDataCommandService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// SaveQuote 保存报价数据
func (s *MarketDataCommandService) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	quote := domain.NewQuote(symbol, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize)
	return s.repo.SaveQuote(ctx, quote)
}

// SaveKline 保存K线数据
func (s *MarketDataCommandService) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return s.repo.SaveKline(ctx, kline)
}

// SaveTrade 保存成交数据
func (s *MarketDataCommandService) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return s.repo.SaveTrade(ctx, trade)
}

// SaveOrderBook 保存订单簿
func (s *MarketDataCommandService) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	return s.repo.SaveOrderBook(ctx, orderBook)
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
