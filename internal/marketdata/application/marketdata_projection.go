package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataProjectionService 负责将行情写模型投影到读模型（Redis）。
type MarketDataProjectionService struct {
	quoteReadRepo     domain.QuoteReadRepository
	klineReadRepo     domain.KlineReadRepository
	tradeReadRepo     domain.TradeReadRepository
	orderBookReadRepo domain.OrderBookReadRepository
	searchRepo        domain.MarketDataSearchRepository
	logger            *slog.Logger
}

func NewMarketDataProjectionService(
	quoteReadRepo domain.QuoteReadRepository,
	klineReadRepo domain.KlineReadRepository,
	tradeReadRepo domain.TradeReadRepository,
	orderBookReadRepo domain.OrderBookReadRepository,
	searchRepo domain.MarketDataSearchRepository,
	logger *slog.Logger,
) *MarketDataProjectionService {
	return &MarketDataProjectionService{
		quoteReadRepo:     quoteReadRepo,
		klineReadRepo:     klineReadRepo,
		tradeReadRepo:     tradeReadRepo,
		orderBookReadRepo: orderBookReadRepo,
		searchRepo:        searchRepo,
		logger:            logger,
	}
}

func (s *MarketDataProjectionService) ProjectQuote(ctx context.Context, quote *domain.Quote) error {
	if s.quoteReadRepo == nil || quote == nil {
		return nil
	}
	if err := s.quoteReadRepo.Save(ctx, quote); err != nil {
		s.logger.WarnContext(ctx, "failed to project quote", "error", err, "symbol", quote.Symbol)
		return err
	}
	if s.searchRepo != nil {
		if err := s.searchRepo.IndexQuote(ctx, quote); err != nil {
			s.logger.WarnContext(ctx, "failed to index quote", "error", err, "symbol", quote.Symbol)
			return err
		}
	}
	return nil
}

func (s *MarketDataProjectionService) ProjectKline(ctx context.Context, kline *domain.Kline) error {
	if s.klineReadRepo == nil || kline == nil {
		return nil
	}
	if err := s.klineReadRepo.Save(ctx, kline); err != nil {
		s.logger.WarnContext(ctx, "failed to project kline", "error", err, "symbol", kline.Symbol, "interval", kline.Interval)
		return err
	}
	return nil
}

func (s *MarketDataProjectionService) ProjectTrade(ctx context.Context, trade *domain.Trade) error {
	if s.tradeReadRepo == nil || trade == nil {
		return nil
	}
	if err := s.tradeReadRepo.Save(ctx, trade); err != nil {
		s.logger.WarnContext(ctx, "failed to project trade", "error", err, "symbol", trade.Symbol, "trade_id", trade.ID)
		return err
	}
	if s.searchRepo != nil {
		if err := s.searchRepo.IndexTrade(ctx, trade); err != nil {
			s.logger.WarnContext(ctx, "failed to index trade", "error", err, "symbol", trade.Symbol, "trade_id", trade.ID)
			return err
		}
	}
	return nil
}

func (s *MarketDataProjectionService) ProjectOrderBook(ctx context.Context, ob *domain.OrderBook) error {
	if s.orderBookReadRepo == nil || ob == nil {
		return nil
	}
	if err := s.orderBookReadRepo.Save(ctx, ob); err != nil {
		s.logger.WarnContext(ctx, "failed to project orderbook", "error", err, "symbol", ob.Symbol)
		return err
	}
	return nil
}
