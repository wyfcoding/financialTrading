package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataService 市场数据门面服务。
type MarketDataService struct {
	Command *MarketDataCommandService
	Query   *MarketDataQueryService
}

// NewMarketDataService 构造函数。
func NewMarketDataService(repo domain.MarketDataRepository, logger *slog.Logger, publisher domain.EventPublisher) *MarketDataService {
	return &MarketDataService{
		Command: NewMarketDataCommandService(repo, logger, publisher),
		Query:   NewMarketDataQueryService(repo),
	}
}

func (s *MarketDataService) SetBroadcaster(b Broadcaster) {
	s.Command.SetBroadcaster(b)
}

// --- Command Facade ---

func (s *MarketDataService) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	return s.Command.HandleTradeExecuted(ctx, event)
}

func (s *MarketDataService) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	return s.Command.SaveQuote(ctx, symbol, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize, timestamp, source)
}

func (s *MarketDataService) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return s.Command.SaveKline(ctx, kline)
}

func (s *MarketDataService) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return s.Command.SaveTrade(ctx, trade)
}

func (s *MarketDataService) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	return s.Command.SaveOrderBook(ctx, orderBook)
}

// --- Query Facade ---

func (s *MarketDataService) GetLatestQuote(ctx context.Context, req *GetLatestQuoteRequest) (*QuoteDTO, error) {
	return s.Query.GetLatestQuote(ctx, req.Symbol)
}

func (s *MarketDataService) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	return s.Query.GetHistoricalQuotes(ctx, symbol, startTime, endTime)
}

func (s *MarketDataService) GetVolatility(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return s.Query.GetVolatility(ctx, symbol)
}

func (s *MarketDataService) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*KlineDTO, error) {
	return s.Query.GetKlines(ctx, symbol, interval, limit)
}

func (s *MarketDataService) GetTrades(ctx context.Context, symbol string, limit int) ([]*TradeDTO, error) {
	return s.Query.GetTrades(ctx, symbol, limit)
}
