package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm/types"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/messagequeue"
)

// MatchingCommandService 处理所有撮合引擎相关的写入操作（Commands）。
type MatchingCommandService struct {
	engine        *domain.DisruptionEngine
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	publisher     messagequeue.EventPublisher
	clearingCli   clearingv1.ClearingServiceClient
	orderCli      orderv1.OrderServiceClient
	logger        *slog.Logger
}

// NewMatchingCommandService 构造函数。
func NewMatchingCommandService(
	symbol string,
	engine *domain.DisruptionEngine,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
	publisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *MatchingCommandService {
	return &MatchingCommandService{
		engine:        engine,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		publisher:     publisher,
		logger:        logger.With("module", "matching_command_service", "symbol", symbol),
	}
}

// StartEngine 启动撮合引擎
func (m *MatchingCommandService) StartEngine() error {
	if m.engine == nil {
		return fmt.Errorf("engine is nil")
	}
	return m.engine.Start()
}

// SetClearingClient 设置清算服务客户端。
func (m *MatchingCommandService) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	m.clearingCli = cli
}

// SetOrderClient 设置订单服务客户端。
func (m *MatchingCommandService) SetOrderClient(cli orderv1.OrderServiceClient) {
	m.orderCli = cli
}

// RecoverState 从数据库恢复引擎状态
func (m *MatchingCommandService) RecoverState(ctx context.Context) error {
	m.logger.Info("starting matching engine state recovery from Order Service", "symbol", m.engine.Symbol())

	if m.orderCli == nil {
		return fmt.Errorf("critical error: order client is nil, cannot recover engine state")
	}

	activeStatuses := []string{"OPEN", "PARTIALLY_FILLED"}
	totalReplayed := 0

	for _, status := range activeStatuses {
		page := int32(1)
		pageSize := int32(500)

		for {
			m.logger.Debug("fetching active orders page", "status", status, "page", page)

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
				break
			}

			for _, o := range resp.Orders {
				price := decimal.NewFromFloat(o.Price)
				qty := decimal.NewFromFloat(o.Quantity)
				filled := decimal.NewFromFloat(o.FilledQuantity)
				remQty := qty.Sub(filled)

				if remQty.IsPositive() {
					m.engine.ReplayOrder(&types.Order{
						OrderID:   o.Id,
						Symbol:    o.Symbol,
						Side:      types.Side(o.Side),
						Price:     price,
						Quantity:  remQty,
						UserID:    o.UserId,
						Timestamp: o.CreatedAt.AsTime().UnixNano(),
					})
					totalReplayed++
				}
			}

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
func (m *MatchingCommandService) SubmitOrder(ctx context.Context, cmd *SubmitOrderCommand) (*domain.MatchingResult, error) {
	if m.engine.IsHalted() {
		return nil, fmt.Errorf("matching engine is currently unavailable (halted)")
	}
	defer logging.LogDuration(ctx, "Order matching processing finished",
		"order_id", cmd.OrderID,
		"symbol", cmd.Symbol,
	)()

	price, err := decimal.NewFromString(cmd.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	quantity, err := decimal.NewFromString(cmd.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	displayQty := decimal.Zero
	if cmd.IsIceberg && cmd.IcebergDisplayQuantity != "" {
		displayQty, _ = decimal.NewFromString(cmd.IcebergDisplayQuantity)
	}

	order := &types.Order{
		OrderID:    cmd.OrderID,
		Symbol:     cmd.Symbol,
		Side:       types.Side(cmd.Side),
		Price:      price,
		Quantity:   quantity,
		UserID:     cmd.UserID,
		IsIceberg:  cmd.IsIceberg,
		DisplayQty: displayQty,
		PostOnly:   cmd.PostOnly,
		Timestamp:  time.Now().UnixNano(),
	}

	m.logger.Debug("submitting order to disruption engine", "order_id", order.OrderID, "side", order.Side, "price", order.Price.String(), "qty", order.Quantity.String())
	result, err := m.engine.SubmitOrder(order)
	if err != nil {
		m.logger.Error("failed to submit order to engine", "order_id", order.OrderID, "error", err)
		return nil, err
	}

	m.logger.Info("order processed by engine", "order_id", order.OrderID, "status", result.Status, "trades_count", len(result.Trades), "remaining_qty", result.RemainingQuantity.String())

	if len(result.Trades) > 0 {
		m.processPostMatching(result.Trades)
	}

	return result, nil
}

// CancelOrder 撤销订单
func (m *MatchingCommandService) CancelOrder(ctx context.Context, orderID string, side string) (*domain.CancelResult, error) {
	m.logger.Info("cancelling order", "order_id", orderID)
	req := &domain.CancelRequest{
		OrderID:   orderID,
		Symbol:    m.engine.Symbol(),
		Side:      types.Side(side),
		Timestamp: time.Now().UnixNano(),
	}
	return m.engine.CancelOrder(req)
}

// BatchSubmitOrder 批量提交订单
func (m *MatchingCommandService) BatchSubmitOrder(ctx context.Context, cmds []*SubmitOrderCommand) ([]*domain.MatchingResult, error) {
	results := make([]*domain.MatchingResult, len(cmds))
	for i, cmd := range cmds {
		res, err := m.SubmitOrder(ctx, cmd)
		if err != nil {
			return nil, err
		}
		results[i] = res
	}
	return results, nil
}

// RunAuction 启动集合竞价
func (m *MatchingCommandService) RunAuction(ctx context.Context) (*domain.AuctionResult, error) {
	m.logger.Info("triggering manual auction execution")
	res, err := m.engine.ExecuteAuction()
	if err != nil {
		return nil, err
	}

	if len(res.Trades) > 0 {
		m.processPostMatching(res.Trades)
	}
	return res, nil
}

func (m *MatchingCommandService) processPostMatching(trades []*types.Trade) {
	m.logger.Debug("starting reliable post-matching processing", "count", len(trades))

	err := m.tradeRepo.WithTx(context.Background(), func(txCtx context.Context) error {
		for _, t := range trades {
			domainTrade := &domain.Trade{
				TradeID:     t.TradeID,
				BuyOrderID:  t.BuyOrderID,
				SellOrderID: t.SellOrderID,
				Symbol:      t.Symbol,
				Price:       t.Price.InexactFloat64(),
				Quantity:    t.Quantity.InexactFloat64(),
				Timestamp:   time.Unix(0, t.Timestamp),
			}
			if err := m.tradeRepo.Save(txCtx, domainTrade); err != nil {
				return fmt.Errorf("failed to persist trade %s: %w", t.TradeID, err)
			}

			if m.publisher != nil {
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
				if err := m.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.TradeExecutedEventType, t.TradeID, event); err != nil {
					return fmt.Errorf("failed to publish outbox event for trade %s: %w", t.TradeID, err)
				}
			}
		}
		return nil
	})

	if err != nil {
		m.logger.Error("CRITICAL: failed post-matching transactional processing. HALTING ENGINE!", "error", err)
		m.engine.Halt()
	} else {
		m.logger.Info("post-matching trades persisted and outbox events created", "count", len(trades))
		m.dispatchSettlement(trades)
	}
}

func (m *MatchingCommandService) dispatchSettlement(trades []*types.Trade) {
	if m.clearingCli == nil || len(trades) == 0 {
		return
	}

	for _, t := range trades {
		req := &clearingv1.SettleTradeRequest{
			TradeId:    t.TradeID,
			BuyUserId:  t.BuyUserID,
			SellUserId: t.SellUserID,
			Symbol:     t.Symbol,
			Quantity:   t.Quantity.String(),
			Price:      t.Price.String(),
			Currency:   settlementCurrency(t.Symbol),
		}

		callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := m.clearingCli.SettleTrade(callCtx, req)
		cancel()
		if err != nil {
			m.logger.Error("failed to trigger settlement for trade",
				"trade_id", t.TradeID,
				"symbol", t.Symbol,
				"error", err)
		}
	}
}

func settlementCurrency(symbol string) string {
	if symbol == "" {
		return "USDT"
	}
	parts := strings.FieldsFunc(symbol, func(r rune) bool {
		return r == '-' || r == '/' || r == '_'
	})
	if len(parts) >= 2 && parts[len(parts)-1] != "" {
		return strings.ToUpper(parts[len(parts)-1])
	}
	return "USDT"
}

// SaveSnapshot 触发快照
func (m *MatchingCommandService) SaveSnapshot(ctx context.Context, depth int) error {
	snapshot := m.engine.GetOrderBookSnapshot(depth)
	if err := m.orderBookRepo.SaveSnapshot(ctx, snapshot); err != nil {
		return err
	}
	if m.publisher != nil {
		event := map[string]any{
			"symbol":    snapshot.Symbol,
			"timestamp": snapshot.Timestamp,
			"depth":     depth,
		}
		return m.publisher.Publish(ctx, domain.OrderBookSnapshotEventType, snapshot.Symbol, event)
	}
	return nil
}
