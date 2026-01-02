package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	clearingv1 "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
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
	orderCli      orderv1.OrderServiceClient
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

func (m *MatchingEngineManager) SetOrderClient(cli orderv1.OrderServiceClient) {
	m.orderCli = cli
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

	displayQty := decimal.Zero
	if req.IsIceberg && req.IcebergDisplayQuantity != "" {
		displayQty, _ = decimal.NewFromString(req.IcebergDisplayQuantity)
	}

	// 封装为算法层订单对象
	order := &algorithm.Order{
		OrderID:    req.OrderID,
		Symbol:     req.Symbol,
		Side:       req.Side,
		Price:      price,
		Quantity:   quantity,
		UserID:     req.UserID,
		IsIceberg:  req.IsIceberg,
		DisplayQty: displayQty,
		PostOnly:   req.PostOnly,
		Timestamp:  time.Now().UnixNano(),
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

			// 2. 报告到清算服务
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
				}
			}

			// 3. 更新订单成交状态 (Cross-Project Interaction)
			if m.orderCli != nil {
				m.logger.Info("reporting fill to order service", "buy_order_id", t.BuyOrderID, "sell_order_id", t.SellOrderID, "qty", t.Quantity.String(), "price", t.Price.String())
				
				// 为买方更新
				_, _ = m.orderCli.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
					OrderId:          t.BuyOrderID,
					UserId:           t.BuyUserID,
					Status:           "PARTIALLY_FILLED",
					FilledQuantity:   t.Quantity.String(),
					LastFillPrice:    t.Price.String(),
					LastFillQuantity: t.Quantity.String(),
					Remark:           fmt.Sprintf("Buy filled via trade %s", t.TradeID),
				})

				// 为卖方更新
				_, _ = m.orderCli.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
					OrderId:          t.SellOrderID,
					UserId:           t.SellUserID,
					Status:           "PARTIALLY_FILLED",
					FilledQuantity:   t.Quantity.String(),
					LastFillPrice:    t.Price.String(),
					LastFillQuantity: t.Quantity.String(),
					Remark:           fmt.Sprintf("Sell filled via trade %s", t.TradeID),
				})
			}
		}
	}()
}

// SaveSnapshot (Manually trigger snapshot if needed)
func (m *MatchingEngineManager) SaveSnapshot(ctx context.Context, depth int) error {
	snapshot := m.engine.GetOrderBookSnapshot(depth)
	return m.orderBookRepo.SaveSnapshot(ctx, snapshot)
}
