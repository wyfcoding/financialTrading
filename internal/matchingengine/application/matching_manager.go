package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/pkg/algorithm/types"

	"github.com/shopspring/decimal"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/contextx"
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

// SetClearingClient 设置清算服务客户端。
func (m *MatchingEngineManager) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	m.clearingCli = cli
}

// SetOrderClient 设置订单服务客户端。
func (m *MatchingEngineManager) SetOrderClient(cli orderv1.OrderServiceClient) {
	m.orderCli = cli
}

// RecoverState 从数据库恢复引擎状态 (生产级分页重放实现)
func (m *MatchingEngineManager) RecoverState(ctx context.Context) error {
	m.logger.Info("starting matching engine state recovery from Order Service", "symbol", m.engine.Symbol())

	if m.orderCli == nil {
		return fmt.Errorf("critical error: order client is nil, cannot recover engine state")
	}

	// 定义需要恢复的活跃状态列表
	// OPEN: 挂单中, PARTIALLY_FILLED: 部分成交
	activeStatuses := []string{"OPEN", "PARTIALLY_FILLED"}
	totalReplayed := 0

	for _, status := range activeStatuses {
		page := int32(1)
		pageSize := int32(500) // 每页拉取 500 条，平衡网络带宽与内存占用

		for {
			m.logger.Debug("fetching active orders page", "status", status, "page", page)

			// 通过 gRPC 调用 Order 服务获取活跃订单
			// 仓库实现中已保证按 CreatedAt 正序排列，确保回放时的时间优先级 (FIFO) 正确
			resp, err := m.orderCli.ListOrders(ctx, &orderv1.ListOrdersRequest{
				Symbol: m.engine.Symbol(),
				Status: status,
				Offset: (page - 1) * pageSize,
				Limit:  pageSize,
			})
			if err != nil {
				return fmt.Errorf("failed to fetch orders from OrderService (status=%s, page=%d): %w", status, page, err)
			}

			if len(resp.Orders) == 0 {
				break // 该状态下的订单已拉取完毕
			}

			for _, o := range resp.Orders {
				// 1. 解析价格与数量
				price := decimal.NewFromFloat(o.Price)
				qty := decimal.NewFromFloat(o.Quantity)
				filled := decimal.NewFromFloat(o.FilledQuantity)

				// 2. 计算剩余可撮合数量 (总数量 - 已成交数量)
				remQty := qty.Sub(filled)

				if remQty.IsPositive() {
					// 3. 将订单无损注入内存订单簿
					// 注意：此处调用 ReplayOrder，它只负责重建索引，不会触发成交或发送任何消息
					m.engine.ReplayOrder(&types.Order{
						OrderID:   o.Id,
						Symbol:    o.Symbol,
						Side:      types.Side(o.Side),
						Price:     price,
						Quantity:  remQty,
						UserID:    o.UserId,
						Timestamp: o.CreatedAt.AsTime().UnixNano(), // 使用原始下单时间戳，保证撮合队列的绝对公平
					})
					totalReplayed++
				}
			}

			// 如果返回结果少于页大小，说明是最后一页
			if int32(len(resp.Orders)) < pageSize {
				break
			}
			page++
		}
	}

	m.logger.Info("matching engine state recovery completed successfully",
		"symbol", m.engine.Symbol(),
		"total_replayed_orders", totalReplayed)
	return nil
}

// SubmitOrder 提交订单进行撮合
func (m *MatchingEngineManager) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	// 入口拦截：若引擎正在恢复中或因故障停止，拒绝新请求
	if m.engine.IsHalted() {
		return nil, fmt.Errorf("matching engine is currently unavailable (halted)")
	}
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
	order := &types.Order{
		OrderID:    req.OrderID,
		Symbol:     req.Symbol,
		Side:       types.Side(req.Side),
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

func (m *MatchingEngineManager) processPostMatching(trades []*types.Trade) {
	m.logger.Debug("starting reliable post-matching processing", "count", len(trades))

	// 使用本地事务确保成交记录与 Outbox 消息的一致性
	err := m.db.Transaction(func(tx *gorm.DB) error {
		// 将事务注入 Context，供 Repository 使用
		ctx := contextx.WithTx(context.Background(), tx)

		for _, t := range trades {
			// 1. 持久化成交记录
			if err := m.tradeRepo.Save(ctx, t); err != nil {
				return fmt.Errorf("failed to persist trade %s: %w", t.TradeID, err)
			}

			// 2. 写入 Outbox 事件
			event := map[string]any{
				"trade_id":      t.TradeID,
				"buy_order_id":  t.BuyOrderID,
				"sell_order_id": t.SellOrderID,
				"buy_user_id":   t.BuyUserID,
				"sell_user_id":  t.SellUserID,
				"symbol":        t.Symbol,
				"quantity":      t.Quantity.String(),
				"price":         t.Price.String(),
				"executed_at":   t.Timestamp,
			}

			if err := m.outbox.PublishInTx(ctx, tx, "trade.executed", t.TradeID, event); err != nil {
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

// SaveSnapshot 触发快照
func (m *MatchingEngineManager) SaveSnapshot(ctx context.Context, depth int) error {
	snapshot := m.engine.GetOrderBookSnapshot(depth)
	return m.orderBookRepo.SaveSnapshot(ctx, snapshot)
}
