package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	clearingv1 "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/logging"
)

// MatchingEngineManager 处理所有撮合引擎相关的写入操作（Commands）。
type MatchingEngineManager struct {
	engine        *domain.DisruptionEngine
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	clearingCli   clearingv1.ClearingServiceClient
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

func (m *MatchingEngineManager) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	m.clearingCli = cli
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

	// 异步持久化成交记录并报告清算
	if len(result.Trades) > 0 {
		m.processPostMatching(result.Trades)
	}

	return result, nil
}

func (m *MatchingEngineManager) processPostMatching(trades []*algorithm.Trade) {
	m.logger.Debug("starting post-matching processing", "count", len(trades))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, t := range trades {
			// 1. 持久化到本地 DB
			if err := m.tradeRepo.Save(ctx, t); err != nil {
				m.logger.Error("failed to persist trade", "trade_id", t.TradeID, "error", err)
			}

			// 2. 报告到清算服务 (Internal Interaction)
			if m.clearingCli != nil {
				_, err := m.clearingCli.SettleTrade(ctx, &clearingv1.SettleTradeRequest{
					TradeId:    t.TradeID,
					BuyUserId:  t.BuyUserID,
					SellUserId: t.SellUserID,
					Symbol:     t.Symbol,
					Quantity:   t.Quantity.String(),
					Price:      t.Price.String(),
				})
				if err != nil {
					m.logger.Error("failed to report trade to clearing", "trade_id", t.TradeID, "error", err)
				} else {
					m.logger.Info("trade reported to clearing successfully", "trade_id", t.TradeID)
				}
			}
		}
	}()
}

// SaveSnapshot (Manually trigger snapshot if needed)
func (m *MatchingEngineManager) SaveSnapshot(ctx context.Context, depth int) error {
	snapshot := m.engine.GetOrderBookSnapshot(depth)
	return m.orderBookRepo.SaveSnapshot(ctx, snapshot)
}
