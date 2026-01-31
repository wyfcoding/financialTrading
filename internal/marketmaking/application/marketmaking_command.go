package application

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingCommandService 做市命令服务
type MarketMakingCommandService struct {
	repo       domain.MarketMakingRepository
	orderSvc   domain.OrderClient
	marketSvc  domain.MarketDataClient
	publisher  domain.EventPublisher
	workers    map[string]context.CancelFunc
}

// NewMarketMakingCommandService 创建做市命令服务实例
func NewMarketMakingCommandService(
	repo domain.MarketMakingRepository,
	orderSvc domain.OrderClient,
	marketSvc domain.MarketDataClient,
	publisher domain.EventPublisher,
) *MarketMakingCommandService {
	return &MarketMakingCommandService{
		repo:       repo,
		orderSvc:   orderSvc,
		marketSvc:  marketSvc,
		publisher:  publisher,
		workers:    make(map[string]context.CancelFunc),
	}
}

// SetStrategy 处理设置做市策略
func (s *MarketMakingCommandService) SetStrategy(ctx context.Context, cmd SetStrategyCommand) (string, error) {
	// Parse fields
	spread, _ := decimal.NewFromString(cmd.Spread)
	minSz, _ := decimal.NewFromString(cmd.MinOrderSize)
	maxSz, _ := decimal.NewFromString(cmd.MaxOrderSize)
	maxPos, _ := decimal.NewFromString(cmd.MaxPosition)

	strategy, err := s.repo.GetStrategyBySymbol(ctx, cmd.Symbol)
	if err != nil {
		return "", err
	}

	isNew := false
	if strategy == nil {
		strategy = domain.NewQuoteStrategy(cmd.Symbol, spread, minSz, maxSz, maxPos)
		isNew = true
	} else {
		strategy.UpdateConfig(spread, minSz, maxSz, maxPos)
	}

	oldStatus := strategy.Status
	if strings.ToUpper(cmd.Status) == "ACTIVE" {
		strategy.Activate()
	} else if strings.ToUpper(cmd.Status) == "PAUSED" {
		strategy.Pause()
	}

	if err := s.repo.SaveStrategy(ctx, strategy); err != nil {
		return "", err
	}

	// 发布策略事件
	if isNew {
		event := domain.StrategyCreatedEvent{
			StrategyID:   strategy.Symbol,
			Symbol:       strategy.Symbol,
			Spread:       strategy.Spread.String(),
			MinOrderSize: strategy.MinOrderSize.String(),
			MaxOrderSize: strategy.MaxOrderSize.String(),
			MaxPosition:  strategy.MaxPosition.String(),
			Status:       string(strategy.Status),
			Timestamp:    time.Now(),
		}
		s.publisher.Publish(ctx, "marketmaking.strategy.created", strategy.Symbol, event)
	} else {
		event := domain.StrategyUpdatedEvent{
			StrategyID:   strategy.Symbol,
			Symbol:       strategy.Symbol,
			Spread:       strategy.Spread.String(),
			MinOrderSize: strategy.MinOrderSize.String(),
			MaxOrderSize: strategy.MaxOrderSize.String(),
			MaxPosition:  strategy.MaxPosition.String(),
			Status:       string(strategy.Status),
			Timestamp:    time.Now(),
		}
		s.publisher.Publish(ctx, "marketmaking.strategy.updated", strategy.Symbol, event)
	}

	// 发布状态变更事件
	if oldStatus != strategy.Status {
		if strategy.Status == domain.StrategyStatusActive {
			event := domain.StrategyActivatedEvent{
				StrategyID: strategy.Symbol,
				Symbol:     strategy.Symbol,
				Timestamp:  time.Now(),
			}
			s.publisher.Publish(ctx, "marketmaking.strategy.activated", strategy.Symbol, event)
		} else if strategy.Status == domain.StrategyStatusPaused {
			event := domain.StrategyPausedEvent{
				StrategyID: strategy.Symbol,
				Symbol:     strategy.Symbol,
				Timestamp:  time.Now(),
			}
			s.publisher.Publish(ctx, "marketmaking.strategy.paused", strategy.Symbol, event)
		}
	}

	// 管理运行工人
	if strategy.Status == domain.StrategyStatusActive {
		s.startWorker(strategy.Symbol)
	} else {
		s.stopWorker(strategy.Symbol)
	}

	return strategy.Symbol, nil
}

// startWorker 启动做市工人
func (s *MarketMakingCommandService) startWorker(symbol string) {
	if _, ok := s.workers[symbol]; ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.workers[symbol] = cancel
	go s.runWorker(ctx, symbol)
}

// stopWorker 停止做市工人
func (s *MarketMakingCommandService) stopWorker(symbol string) {
	if cancel, ok := s.workers[symbol]; ok {
		cancel()
		delete(s.workers, symbol)
	}
}

// runWorker 运行做市工人
func (s *MarketMakingCommandService) runWorker(ctx context.Context, symbol string) {
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
			quotes := model.CalculateQuotes(midPrice, inventory)
			bid := quotes.BidPrice
			ask := quotes.AskPrice

			// 3. 下单 (简化的做市演示：先撤旧单再下新单，此处略去撤单逻辑)
			qty := decimal.NewFromFloat(1.0) // 演示固定数量

			// 下单买盘
			orderID, err := s.orderSvc.PlaceOrder(ctx, symbol, "BUY", bid, qty)
			if err == nil {
				// 发布做市报价下单事件
				event := domain.MarketMakingQuotePlacedEvent{
					StrategyID: symbol,
					Symbol:     symbol,
					Side:       "BUY",
					Price:      bid.String(),
					Quantity:   qty.String(),
					OrderID:    orderID,
					Timestamp:  time.Now(),
				}
				s.publisher.Publish(ctx, "marketmaking.quote.placed", symbol, event)
			}

			// 下单卖盘
			orderID, err = s.orderSvc.PlaceOrder(ctx, symbol, "SELL", ask, qty)
			if err == nil {
				// 发布做市报价下单事件
				event := domain.MarketMakingQuotePlacedEvent{
					StrategyID: symbol,
					Symbol:     symbol,
					Side:       "SELL",
					Price:      ask.String(),
					Quantity:   qty.String(),
					OrderID:    orderID,
					Timestamp:  time.Now(),
				}
				s.publisher.Publish(ctx, "marketmaking.quote.placed", symbol, event)
			}
		}
	}
}
