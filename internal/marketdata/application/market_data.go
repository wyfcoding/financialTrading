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

func (s *MarketDataService) GetLatestQuote(ctx context.Context, req *GetLatestQuoteRequest) (*QuoteDTO, error) {
	return s.query.GetLatestQuote(ctx, req.Symbol)
}

func (s *MarketDataService) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	return s.query.GetHistoricalQuotes(ctx, symbol, startTime, endTime)
}

func (s *MarketDataService) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*KlineDTO, error) {
	return s.query.GetKlines(ctx, symbol, interval, limit)
}

func (s *MarketDataService) GetTrades(ctx context.Context, symbol string, limit int) ([]*TradeDTO, error) {
	return s.query.GetTrades(ctx, symbol, limit)
}

// Broadcaster 广播接口
type Broadcaster interface {
	Broadcast(topic string, data any) error
}

// MarketDataManager 负责行情数据的维护与事件触发。
type MarketDataManager struct {
	quoteRepo     domain.QuoteRepository
	klineRepo     domain.KlineRepository
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	logger        *slog.Logger
	broadcaster   Broadcaster
}

func NewMarketDataManager(qr domain.QuoteRepository, kr domain.KlineRepository, tr domain.TradeRepository, obr domain.OrderBookRepository, logger *slog.Logger) *MarketDataManager {
	return &MarketDataManager{
		quoteRepo:     qr,
		klineRepo:     kr,
		tradeRepo:     tr,
		orderBookRepo: obr,
		logger:        logger,
	}
}

func (m *MarketDataManager) SetBroadcaster(b Broadcaster) {
	m.broadcaster = b
}

func (m *MarketDataManager) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	// 实现成交事件后的行情更新逻辑（如更新 Kline）
	return nil
}

func (m *MarketDataManager) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	quote := domain.NewQuote(symbol, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize)
	return m.quoteRepo.Save(ctx, quote)
}

func (m *MarketDataManager) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return m.klineRepo.Save(ctx, kline)
}

func (m *MarketDataManager) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return m.tradeRepo.Save(ctx, trade)
}

func (m *MarketDataManager) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	return m.orderBookRepo.Save(ctx, orderBook)
}

// MarketDataQuery 负责行情数据的查询。
type MarketDataQuery struct {
	quoteRepo     domain.QuoteRepository
	klineRepo     domain.KlineRepository
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
}

func NewMarketDataQuery(qr domain.QuoteRepository, kr domain.KlineRepository, tr domain.TradeRepository, obr domain.OrderBookRepository) *MarketDataQuery {
	return &MarketDataQuery{
		quoteRepo:     qr,
		klineRepo:     kr,
		tradeRepo:     tr,
		orderBookRepo: obr,
	}
}

func (q *MarketDataQuery) GetLatestQuote(ctx context.Context, symbol string) (*QuoteDTO, error) {
	quote, err := q.quoteRepo.GetLatest(ctx, symbol)
	if err != nil || quote == nil {
		return nil, err
	}
	return &QuoteDTO{
		Symbol:    quote.Symbol,
		BidPrice:  quote.BidPrice.String(),
		AskPrice:  quote.AskPrice.String(),
		BidSize:   quote.BidSize.String(),
		AskSize:   quote.AskSize.String(),
		LastPrice: quote.LastPrice.String(),
		LastSize:  quote.LastSize.String(),
		Timestamp: quote.Timestamp.UnixMilli(),
	}, nil
}

func (q *MarketDataQuery) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	// 实现查询历史行情逻辑
	return nil, nil
}

func (q *MarketDataQuery) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*KlineDTO, error) {
	klines, err := q.klineRepo.GetKlines(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}
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
	return dtos, nil
}

func (q *MarketDataQuery) GetTrades(ctx context.Context, symbol string, limit int) ([]*TradeDTO, error) {
	trades, err := q.tradeRepo.GetTrades(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
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
	return dtos, nil
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
