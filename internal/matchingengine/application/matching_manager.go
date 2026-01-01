package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/logging"
)

// MatchingEngineManager 处理所有撮合引擎相关的写入操作（Commands）。
type MatchingEngineManager struct {
	engine        *domain.DisruptionEngine
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	logger        *slog.Logger
}

// NewMatchingEngineManager 构造函数。
func NewMatchingEngineManager(symbol string, engine *domain.DisruptionEngine, tradeRepo domain.TradeRepository, orderBookRepo domain.OrderBookRepository, logger *slog.Logger) *MatchingEngineManager {
	return &MatchingEngineManager{
		engine:        engine,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		logger:        logger.With("module", "matching_engine_manager", "symbol", symbol),
	}
}

// SubmitOrder 提交订单进行撮合
func (m *MatchingEngineManager) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	// 记录性能监控
	defer logging.LogDuration(ctx, "Order matching processing finished",
		"order_id", req.OrderID,
		"symbol", req.Symbol,
	)()

	// 验证与解析
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 封装为算法层订单对象
	order := &algorithm.Order{
		OrderID:   req.OrderID,
		Symbol:    req.Symbol,
		Side:      req.Side,
		Price:     price,
		Quantity:  quantity,
		Timestamp: time.Now().UnixNano(),
	}

	// 提交至 Disruptor 引擎
	m.logger.Debug("submitting order to disruption engine", "order_id", order.OrderID, "side", order.Side, "price", order.Price.String(), "qty", order.Quantity.String())
	result, err := m.engine.SubmitOrder(order)
	if err != nil {
		m.logger.Error("failed to submit order to engine", "order_id", order.OrderID, "error", err)
		return nil, err
	}

	m.logger.Info("order processed by engine", "order_id", order.OrderID, "status", result.Status, "trades_count", len(result.Trades), "remaining_qty", result.RemainingQuantity.String())

	// 异步持久化成交记录
	if len(result.Trades) > 0 {
		m.asyncPersistTrades(result.Trades)
	}

	return result, nil
}

func (m *MatchingEngineManager) asyncPersistTrades(trades []*algorithm.Trade) {
	m.logger.Debug("starting async persistence of trades", "count", len(trades))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, t := range trades {
			if err := m.tradeRepo.Save(ctx, t); err != nil {
				m.logger.Error("failed to persist trade", "trade_id", t.TradeID, "error", err)
			}
		}
	}()
}

// SaveSnapshot (Manually trigger snapshot if needed)
func (m *MatchingEngineManager) SaveSnapshot(ctx context.Context, depth int) error {
	snapshot := m.engine.GetOrderBookSnapshot(depth)
	return m.orderBookRepo.SaveSnapshot(ctx, snapshot)
}
