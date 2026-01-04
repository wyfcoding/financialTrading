package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataService 市场数据门面服务，整合 Manager 和 Query。
type MarketDataService struct {
	manager *MarketDataManager
	query   *MarketDataQuery
}

// NewMarketDataService 构造函数。
func NewMarketDataService(
	quoteRepo domain.QuoteRepository,
	klineRepo domain.KlineRepository,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
	logger *slog.Logger,
) *MarketDataService {
	manager := NewMarketDataManager(quoteRepo, klineRepo, tradeRepo, orderBookRepo, logger)
	return &MarketDataService{
		manager: manager,
		query:   NewMarketDataQuery(quoteRepo, klineRepo, tradeRepo, orderBookRepo),
	}
}

func (s *MarketDataService) SetBroadcaster(b Broadcaster) {
	s.manager.SetBroadcaster(b)
}

// --- Manager (Writes) ---

func (s *MarketDataService) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	return s.manager.HandleTradeExecuted(ctx, event)
}

func (s *MarketDataService) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	return s.manager.SaveQuote(ctx, symbol, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize, timestamp, source)
}

func (s *MarketDataService) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return s.manager.SaveKline(ctx, kline)
}

func (s *MarketDataService) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return s.manager.SaveTrade(ctx, trade)
}

func (s *MarketDataService) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	return s.manager.SaveOrderBook(ctx, orderBook)
}

// --- Query (Reads) ---

func (s *MarketDataService) GetLatestQuote(ctx context.Context, req *GetLatestQuoteRequest) (*QuoteDTO, error) {
	return s.query.GetLatestQuote(ctx, req.Symbol)
}

func (s *MarketDataService) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	return s.query.GetHistoricalQuotes(ctx, symbol, startTime, endTime)
}

// --- Legacy Compatibility Types ---

// GetLatestQuoteRequest 获取最新行情请求 DTO
type GetLatestQuoteRequest struct {
	Symbol string
}

// QuoteDTO 行情数据 DTO
type QuoteDTO struct {
	Symbol    string
	BidPrice  string
	AskPrice  string
	BidSize   string
	AskSize   string
	LastPrice string
	LastSize  string
	Timestamp int64
	Source    string
}
