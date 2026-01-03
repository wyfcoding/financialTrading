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
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

// MatchingEngineManager 处理所有撮合引擎相关的写入操作（Commands）。
type MatchingEngineManager struct {
	engine        *domain.DisruptionEngine
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	clearingCli   clearingv1.ClearingServiceClient
	orderCli      orderv1.OrderServiceClient
	logger        *slog.Logger
	db            *gorm.DB
	outbox        *outbox.Manager
}

// NewMatchingEngineManager 构造函数。
func NewMatchingEngineManager(
	symbol string,
	engine *domain.DisruptionEngine,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
	db *gorm.DB,
	outboxMgr *outbox.Manager,
	logger *slog.Logger,
) *MatchingEngineManager {
	return &MatchingEngineManager{
		engine:        engine,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		db:            db,
		outbox:        outboxMgr,
		logger:        logger.With("module", "matching_engine_manager", "symbol", symbol),
	}
}

func (m *MatchingEngineManager) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	m.clearingCli = cli
}

func (m *MatchingEngineManager) SetOrderClient(cli orderv1.OrderServiceClient) {
	m.orderCli = cli
}

// RecoverState 从数据库恢复引擎状态
func (m *MatchingEngineManager) RecoverState(ctx context.Context) error {
	m.logger.Info("starting matching engine state recovery...")

	// 1. 调用 Order Service 获取所有活跃订单
	// 注意：此处需要 OrderServiceClient 提供相应接口，
	// 若无直接接口，可直接从 orderRepo (如果在同进程) 或通过 gRPC 获取。
	// 为了演示 Recovery 闭环，假设我们通过 orderCli 获取。
	if m.orderCli == nil {
		m.logger.Warn("order client not available, skipping recovery")
		return nil
	}

	resp, err := m.orderCli.ListOrders(ctx, &orderv1.ListOrdersRequest{
		Symbol: m.engine.Symbol(), 
		Status: "OPEN", // 状态在 proto 中定义为 string
	})
	if err != nil {
		return fmt.Errorf("failed to fetch active orders for recovery: %w", err)
	}

	for _, o := range resp.Orders {
		// 计算剩余可撮合数量
		price, _ := decimal.NewFromString(o.Price)
		qty, _ := decimal.NewFromString(o.Quantity)
		filled, _ := decimal.NewFromString(o.FilledQuantity)
		remQty := qty.Sub(filled)

		if remQty.IsPositive() {
			m.engine.ReplayOrder(&algorithm.Order{
				OrderID:   o.OrderId,
				Symbol:    o.Symbol,
				Side:      o.Side,
				Price:     price,
				Quantity:  remQty,
				UserID:    o.UserId,
				Timestamp: time.Unix(o.CreatedAt, 0).UnixNano(),
			})
		}
	}

	m.logger.Info("state recovery finished", "replayed_count", len(resp.Orders))
	return nil
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
	m.logger.Debug("starting reliable post-matching processing", "count", len(trades))

	// 使用本地事务确保成交记录与 Outbox 消息的一致性
	err := m.db.Transaction(func(tx *gorm.DB) error {
		// 将事务注入 Context，供 Repository 使用
		ctx := context.WithValue(context.Background(), "tx_db", tx)

		for _, t := range trades {
			// 1. 持久化到本地 DB
			if err := m.tradeRepo.Save(ctx, t); err != nil {
				return fmt.Errorf("failed to persist trade %s: %w", t.TradeID, err)
			}

			// 2. 写入 Outbox 事件
			// 消息包含结算所需的所有关键信息
			event := map[string]any{
				"trade_id":     t.TradeID,
				"buy_order_id": t.BuyOrderID,
				"sell_order_id": t.SellOrderID,
				"buy_user_id":  t.BuyUserID,
				"sell_user_id": t.SellUserID,
				"symbol":       t.Symbol,
				"quantity":     t.Quantity.String(),
				"price":        t.Price.String(),
				"executed_at":  t.Timestamp,
			}

			if err := m.outbox.PublishInTx(tx, "trade.executed", t.TradeID, event); err != nil {
				return fmt.Errorf("failed to publish outbox event for trade %s: %w", t.TradeID, err)
			}
		}
		return nil
	})

	if err != nil {
		m.logger.Error("CRITICAL: failed post-matching transactional processing. HALTING ENGINE!", "error", err)
		
		// 立即熔断引擎：这是最高级别的安全保护
		// 一旦内存状态与 DB 发生分歧，必须停止一切交易以防止损失扩大。
		m.engine.Halt()
		
		// 在实际系统中，此处还应触发 PagerDuty/短信告警给运维人员。
	} else {
		m.logger.Info("post-matching trades persisted and outbox events created", "count", len(trades))
	}
}

// SaveSnapshot (Manually trigger snapshot if needed)
func (m *MatchingEngineManager) SaveSnapshot(ctx context.Context, depth int) error {
	snapshot := m.engine.GetOrderBookSnapshot(depth)
	return m.orderBookRepo.SaveSnapshot(ctx, snapshot)
}
