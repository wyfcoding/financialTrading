package application

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

type MarketMakingApplicationService struct {
	repo      domain.QuoteStrategyRepository
	orderSvc  domain.OrderClient
	marketSvc domain.MarketDataClient
	workers   map[string]context.CancelFunc // symbol -> stop func
}

func NewMarketMakingApplicationService(repo domain.QuoteStrategyRepository, orderSvc domain.OrderClient, marketSvc domain.MarketDataClient) *MarketMakingApplicationService {
	return &MarketMakingApplicationService{
		repo:      repo,
		orderSvc:  orderSvc,
		marketSvc: marketSvc,
		workers:   make(map[string]context.CancelFunc),
	}
}

func (s *MarketMakingApplicationService) SetStrategy(ctx context.Context, cmd SetStrategyCommand) (string, error) {
	// Parse fields
	spread, _ := decimal.NewFromString(cmd.Spread)
	minSz, _ := decimal.NewFromString(cmd.MinOrderSize)
	maxSz, _ := decimal.NewFromString(cmd.MaxOrderSize)
	maxPos, _ := decimal.NewFromString(cmd.MaxPosition)

	strategy, err := s.repo.GetStrategyBySymbol(ctx, cmd.Symbol)
	if err != nil {
		return "", err
	}

	if strategy == nil {
		strategy = domain.NewQuoteStrategy(cmd.Symbol, spread, minSz, maxSz, maxPos)
	} else {
		strategy.UpdateConfig(spread, minSz, maxSz, maxPos)
	}

	if strings.ToUpper(cmd.Status) == "ACTIVE" {
		strategy.Activate()
	} else if strings.ToUpper(cmd.Status) == "PAUSED" {
		strategy.Pause()
	}

	if err := s.repo.SaveStrategy(ctx, strategy); err != nil {
		return "", err
	}

	// 管理运行工人
	if strategy.Status == domain.StrategyStatusActive {
		s.startWorker(strategy.Symbol)
	} else {
		s.stopWorker(strategy.Symbol)
	}

	return strategy.Symbol, nil // Using Symbol as ID for now
}

func (s *MarketMakingApplicationService) startWorker(symbol string) {
	if _, ok := s.workers[symbol]; ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.workers[symbol] = cancel
	go s.runWorker(ctx, symbol)
}

func (s *MarketMakingApplicationService) stopWorker(symbol string) {
	if cancel, ok := s.workers[symbol]; ok {
		cancel()
		delete(s.workers, symbol)
	}
}

func (s *MarketMakingApplicationService) GetStrategy(ctx context.Context, symbol string) (*StrategyDTO, error) {
	strategy, err := s.repo.GetStrategyBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
	}

	return &StrategyDTO{
		ID:           strategy.Symbol,
		Symbol:       strategy.Symbol,
		Spread:       strategy.Spread.String(),
		MinOrderSize: strategy.MinOrderSize.String(),
		MaxOrderSize: strategy.MaxOrderSize.String(),
		MaxPosition:  strategy.MaxPosition.String(),
		Status:       string(strategy.Status),
		CreatedAt:    strategy.CreatedAt.UnixMilli(),
		UpdatedAt:    strategy.UpdatedAt.UnixMilli(),
	}, nil
}

func (s *MarketMakingApplicationService) GetPerformance(ctx context.Context, symbol string) (*PerformanceDTO, error) {
	// Mock implementation for now
	return &PerformanceDTO{
		Symbol:      symbol,
		TotalPnL:    1023.50,
		TotalVolume: 50000,
		TotalTrades: 125,
		SharpeRatio: 1.8,
	}, nil
}

func (s *MarketMakingApplicationService) runWorker(ctx context.Context, symbol string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 演示用固定参数
	model := domain.NewAvellanedaStoikovModel(0.1, 0.02, 1.5)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 1. 获取中间价和持仓
			midPrice, err := s.marketSvc.GetPrice(ctx, symbol)
			if err != nil {
				continue
			}
			inventory, err := s.orderSvc.GetPosition(ctx, symbol)
			if err != nil {
				inventory = decimal.Zero
			}

			// 2. 计算最优报价
			bid, ask := model.CalculateQuotes(midPrice, inventory).BidPrice, model.CalculateQuotes(midPrice, inventory).AskPrice

			// 3. 下单 (简化的做市演示：先撤旧单再下新单，此处略去撤单逻辑)
			qty := decimal.NewFromFloat(1.0) // 演示固定数量
			s.orderSvc.PlaceOrder(ctx, symbol, "BUY", bid, qty)
			s.orderSvc.PlaceOrder(ctx, symbol, "SELL", ask, qty)
		}
	}
}
