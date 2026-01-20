package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/metrics"
	"gorm.io/gorm"
)

type ExecutionApplicationService struct {
	tradeRepo      domain.TradeRepository
	algoRepo       domain.AlgoOrderRepository
	orderClient    orderv1.OrderServiceClient
	marketData     domain.MarketDataProvider
	volumeProvider domain.VolumeProfileProvider
	metrics        *metrics.Metrics
	outbox         *outbox.Manager
	db             *gorm.DB
}

func NewExecutionApplicationService(
	tradeRepo domain.TradeRepository,
	algoRepo domain.AlgoOrderRepository,
	orderClient orderv1.OrderServiceClient,
	marketData domain.MarketDataProvider,
	volumeProvider domain.VolumeProfileProvider,
	metrics *metrics.Metrics,
	outbox *outbox.Manager,
	db *gorm.DB,
) *ExecutionApplicationService {
	return &ExecutionApplicationService{
		tradeRepo:      tradeRepo,
		algoRepo:       algoRepo,
		orderClient:    orderClient,
		marketData:     marketData,
		volumeProvider: volumeProvider,
		metrics:        metrics,
		outbox:         outbox,
		db:             db,
	}
}

// ExecuteOrder 简单市价/限价成交模拟 (真实情况是对接交易所网关)
func (s *ExecutionApplicationService) ExecuteOrder(ctx context.Context, cmd ExecuteOrderCommand) (*ExecutionDTO, error) {
	// 模拟撮合成功
	tradeID := fmt.Sprintf("TRD-%d", idgen.GenID())
	trade := domain.NewTrade(
		tradeID,
		cmd.OrderID,
		cmd.UserID,
		cmd.Symbol,
		domain.TradeSide(cmd.Side),
		cmd.Price,
		cmd.Quantity,
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		if err := s.tradeRepo.Save(txCtx, trade); err != nil {
			return err
		}

		// 发布 TradeExecuted 事件 (给 Clearing Service 消费)
		return s.outbox.PublishInTx(ctx, tx, "trade.executed", trade.ID, map[string]any{
			"trade_id": trade.ID,
			"order_id": trade.OrderID,
			"symbol":   trade.Symbol,
			"quantity": trade.ExecutedQuantity.String(),
			"price":    trade.ExecutedPrice.String(),
			"user_id":  trade.UserID,
		})
	})
	if err != nil {
		return nil, err
	}

	return &ExecutionDTO{
		ExecutionID: trade.ID,
		OrderID:     trade.OrderID,
		Status:      "FILLED",
		ExecutedQty: trade.ExecutedQuantity.String(),
		ExecutedPx:  trade.ExecutedPrice.String(),
		Timestamp:   trade.ExecutedAt.Unix(),
	}, nil
}

// SubmitAlgoOrder 提交算法订单
func (s *ExecutionApplicationService) SubmitAlgoOrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
	algoID := fmt.Sprintf("ALGO-%d", idgen.GenID())
	start := time.Unix(cmd.StartTime, 0)
	end := time.Unix(cmd.EndTime, 0)

	algoOrder := domain.NewAlgoOrder(
		algoID,
		cmd.UserID,
		cmd.Symbol,
		domain.TradeSide(cmd.Side),
		cmd.TotalQty,
		domain.AlgoType(cmd.AlgoType),
		start,
		end,
		cmd.Params,
	)

	if err := s.algoRepo.Save(ctx, algoOrder); err != nil {
		return "", err
	}

	return algoID, nil
}

// SubmitSOROrder 提交智能路由订单
// SubmitSOROrder 提交智能路由订单
func (s *ExecutionApplicationService) SubmitSOROrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
	// 真实 SOR 逻辑：直接分片并提交至 OrderService (或创建母单进行异步管理)
	// 这里演示同步分片并立即提交
	algoID := fmt.Sprintf("SOR-%d", idgen.GenID())
	order := domain.NewAlgoOrder(algoID, cmd.UserID, cmd.Symbol, domain.TradeSide(cmd.Side), cmd.TotalQty, domain.AlgoTypeSOR, time.Now(), time.Now().Add(time.Hour), cmd.Params)

	strategy := &domain.SORStrategy{
		Provider: s.marketData,
		Venues: []*domain.Venue{
			{ID: "MAIN", Name: "Main Exchange", ExecutionFee: decimal.Zero, Latency: 0, Weight: 1.0},
		},
	}
	slices, err := strategy.GenerateSlices(order)
	if err != nil {
		return "", err
	}

	for _, slice := range slices {
		// 调用 OrderService 提交子单
		var side orderv1.OrderSide
		if order.Side == domain.TradeSideBuy {
			side = orderv1.OrderSide_BUY
		} else {
			side = orderv1.OrderSide_SELL
		}

		var otype orderv1.OrderType
		if slice.OrderType == "MARKET" {
			otype = orderv1.OrderType_MARKET
		} else {
			otype = orderv1.OrderType_LIMIT
		}

		_, _ = s.orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
			UserId:   order.UserID,
			Symbol:   order.Symbol,
			Side:     side,
			Type:     otype,
			Price:    slice.Price.InexactFloat64(),
			Quantity: slice.Quantity.InexactFloat64(),
		})
	}

	return algoID, nil
}

// SubmitFIXOrder 处理来自 FIX 网关的订单请求
func (s *ExecutionApplicationService) SubmitFIXOrder(ctx context.Context, cmd SubmitFIXOrderCommand) (*ExecutionDTO, error) {
	// 简单实现：将 FIX 订单转换为直接执行
	executeCmd := ExecuteOrderCommand{
		OrderID:  cmd.ClOrdID, // FIX 客户端提供的 ID
		UserID:   cmd.UserID,
		Symbol:   cmd.Symbol,
		Side:     cmd.Side,
		Price:    cmd.Price,
		Quantity: cmd.Quantity,
	}
	return s.ExecuteOrder(ctx, executeCmd)
}

// StartAlgoWorker 启动母单执行背景工作线程
func (s *ExecutionApplicationService) StartAlgoWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processActiveAlgoOrders(ctx)
		}
	}
}

func (s *ExecutionApplicationService) processActiveAlgoOrders(ctx context.Context) {
	orders, err := s.algoRepo.ListActive(ctx)
	if err != nil {
		return
	}

	for _, order := range orders {
		var strategy domain.ExecutionStrategy
		switch order.AlgoType {
		case domain.AlgoTypeTWAP:
			strategy = &domain.TWAPStrategy{
				MinSliceSize: order.TotalQuantity.Div(decimal.NewFromInt(10)), // 简单切10片
				Randomize:    true,
			}
		case domain.AlgoTypeVWAP:
			strategy = &domain.VWAPStrategy{
				VolumeProfileProvider: s.volumeProvider,
			}
		case domain.AlgoTypePOV:
			strategy = &domain.POVStrategy{
				Provider: s.marketData,
			}
		case domain.AlgoTypeSOR:
			strategy = &domain.SORStrategy{
				Provider: s.marketData,
				Venues: []*domain.Venue{
					{ID: "MAIN", Name: "Main Exchange", ExecutionFee: decimal.Zero, Latency: 0, Weight: 1.0},
				},
			}
		default:
			continue
		}

		slices, err := strategy.GenerateSlices(order)
		if err != nil || len(slices) == 0 {
			continue
		}

		for _, slice := range slices {
			var side orderv1.OrderSide
			if order.Side == domain.TradeSideBuy {
				side = orderv1.OrderSide_BUY
			} else {
				side = orderv1.OrderSide_SELL
			}

			var otype orderv1.OrderType
			if slice.OrderType == "MARKET" {
				otype = orderv1.OrderType_MARKET
			} else {
				otype = orderv1.OrderType_LIMIT
			}

			_, err := s.orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
				UserId:   order.UserID,
				Symbol:   order.Symbol,
				Side:     side,
				Type:     otype,
				Price:    slice.Price.InexactFloat64(),
				Quantity: slice.Quantity.InexactFloat64(),
			})
			if err == nil {
				order.ExecutedQuantity = order.ExecutedQuantity.Add(slice.Quantity)
			}
		}

		if order.ExecutedQuantity.GreaterThanOrEqual(order.TotalQuantity) {
			order.Status = "COMPLETED"
		}
		_ = s.algoRepo.Save(ctx, order)
	}
}

// GetExecutionHistory 获取执行历史
func (s *ExecutionApplicationService) GetExecutionHistory(ctx context.Context, userID string, limit, offset int) ([]*ExecutionDTO, int64, error) {
	trades, err := s.tradeRepo.List(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*ExecutionDTO, 0, len(trades))
	for _, t := range trades {
		dtos = append(dtos, &ExecutionDTO{
			ExecutionID: t.ID,
			OrderID:     t.OrderID,
			Symbol:      t.Symbol,
			Status:      "FILLED",
			ExecutedQty: t.ExecutedQuantity.String(),
			ExecutedPx:  t.ExecutedPrice.String(),
			Timestamp:   t.ExecutedAt.Unix(),
		})
	}
	return dtos, int64(len(dtos)), nil
}
