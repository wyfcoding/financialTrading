package application

import (
	"context"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataQueryService 处理所有市场数据查询操作（Queries）。
type MarketDataQueryService struct {
	repo              domain.MarketDataRepository
	quoteReadRepo     domain.QuoteReadRepository
	klineReadRepo     domain.KlineReadRepository
	tradeReadRepo     domain.TradeReadRepository
	orderBookReadRepo domain.OrderBookReadRepository
	searchRepo        domain.MarketDataSearchRepository
	history           *HistoryService
}

// NewMarketDataQueryService 构造函数。
func NewMarketDataQueryService(
	repo domain.MarketDataRepository,
	quoteReadRepo domain.QuoteReadRepository,
	klineReadRepo domain.KlineReadRepository,
	tradeReadRepo domain.TradeReadRepository,
	orderBookReadRepo domain.OrderBookReadRepository,
	searchRepo domain.MarketDataSearchRepository,
	history *HistoryService,
) *MarketDataQueryService {
	return &MarketDataQueryService{
		repo:              repo,
		quoteReadRepo:     quoteReadRepo,
		klineReadRepo:     klineReadRepo,
		tradeReadRepo:     tradeReadRepo,
		orderBookReadRepo: orderBookReadRepo,
		searchRepo:        searchRepo,
		history:           history,
	}
}

// GetLatestQuote 获取最新报价
func (s *MarketDataQueryService) GetLatestQuote(ctx context.Context, symbol string) (*QuoteDTO, error) {
	if s.quoteReadRepo != nil {
		if cached, err := s.quoteReadRepo.GetLatest(ctx, symbol); err == nil && cached != nil {
			return toQuoteDTO(cached), nil
		}
	}

	quote, err := s.repo.GetLatestQuote(ctx, symbol)
	if err != nil || quote == nil {
		return nil, err
	}
	if s.quoteReadRepo != nil {
		_ = s.quoteReadRepo.Save(ctx, quote)
	}
	return toQuoteDTO(quote), nil
}

// GetKlines 获取K线数据
func (s *MarketDataQueryService) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*KlineDTO, error) {
	if s.klineReadRepo != nil {
		if cached, err := s.klineReadRepo.List(ctx, symbol, interval, limit); err == nil && len(cached) > 0 {
			return toKlineDTOs(cached), nil
		}
	}

	klines, err := s.repo.GetKlines(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	if s.klineReadRepo != nil {
		for _, k := range klines {
			_ = s.klineReadRepo.Save(ctx, k)
		}
	}
	return toKlineDTOs(klines), nil
}

// GetTrades 获取成交数据
func (s *MarketDataQueryService) GetTrades(ctx context.Context, symbol string, limit int) ([]*TradeDTO, error) {
	if s.tradeReadRepo != nil {
		if cached, err := s.tradeReadRepo.List(ctx, symbol, limit); err == nil && len(cached) > 0 {
			return toTradeDTOs(cached), nil
		}
	}

	if s.searchRepo != nil {
		trades, _, err := s.searchRepo.SearchTrades(ctx, symbol, time.Time{}, time.Time{}, limit, 0)
		if err == nil && len(trades) > 0 {
			return toTradeDTOs(trades), nil
		}
	}

	trades, err := s.repo.GetTrades(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
	if s.tradeReadRepo != nil {
		for _, t := range trades {
			_ = s.tradeReadRepo.Save(ctx, t)
		}
	}
	return toTradeDTOs(trades), nil
}

// GetVolatility 计算波动率
func (s *MarketDataQueryService) GetVolatility(ctx context.Context, symbol string) (decimal.Decimal, error) {
	const interval = "1h"
	const periods = 24
	klines, err := s.repo.GetKlines(ctx, symbol, interval, periods)
	if err != nil || len(klines) < 2 {
		return s.estimateVolatilityFromHistory(ctx, symbol), nil
	}
	return decimal.NewFromFloat(0.25), nil
}

func (s *MarketDataQueryService) GetOrderBook(ctx context.Context, symbol string) (*OrderBookDTO, error) {
	if s.orderBookReadRepo != nil {
		if cached, err := s.orderBookReadRepo.Get(ctx, symbol); err == nil && cached != nil {
			return toOrderBookDTO(cached), nil
		}
	}

	ob, err := s.repo.GetOrderBook(ctx, symbol)
	if err != nil || ob == nil {
		return nil, err
	}
	if s.orderBookReadRepo != nil {
		_ = s.orderBookReadRepo.Save(ctx, ob)
	}
	return toOrderBookDTO(ob), nil
}

// GetHistoricalQuotes 获取历史报价
func (s *MarketDataQueryService) GetHistoricalQuotes(ctx context.Context, symbol string, startTime, endTime int64) ([]*QuoteDTO, error) {
	if s.searchRepo == nil {
		return nil, nil
	}
	var start, end time.Time
	if startTime > 0 {
		start = time.UnixMilli(startTime)
	}
	if endTime > 0 {
		end = time.UnixMilli(endTime)
	}
	quotes, _, err := s.searchRepo.SearchQuotes(ctx, symbol, start, end, 1000, 0)
	if err != nil {
		return nil, err
	}
	return toQuoteDTOs(quotes), nil
}

func (s *MarketDataQueryService) estimateVolatilityFromHistory(ctx context.Context, symbol string) decimal.Decimal {
	base := decimal.NewFromFloat(0.2)
	if s.history == nil {
		return base
	}
	quote, _ := s.repo.GetLatestQuote(ctx, symbol)
	if quote == nil || quote.LastPrice.IsZero() {
		return base
	}

	band := quote.LastPrice.Mul(decimal.NewFromFloat(0.01)) // 1% 价格带
	low := quote.LastPrice.Sub(band)
	high := quote.LastPrice.Add(band)
	count := s.history.QueryVolumeAtTime(symbol, time.Now(), low, high)
	if count <= 0 {
		return base
	}

	adj := 1.0 / math.Sqrt(float64(count))
	vol := base.Mul(decimal.NewFromFloat(adj))
	if vol.LessThan(decimal.NewFromFloat(0.05)) {
		vol = decimal.NewFromFloat(0.05)
	}
	if vol.GreaterThan(decimal.NewFromFloat(1.0)) {
		vol = decimal.NewFromFloat(1.0)
	}
	return vol
}
