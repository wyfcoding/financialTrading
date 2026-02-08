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
	"github.com/wyfcoding/pkg/messagequeue"
)

// ExecutionCommandService 处理所有执行相关的写入操作（Commands）。
type ExecutionCommandService struct {
	tradeRepo      domain.TradeRepository
	algoRepo       domain.AlgoOrderRepository
	redisRepo      domain.AlgoRedisRepository
	eventStore     domain.EventStore
	publisher      messagequeue.EventPublisher
	orderClient    orderv1.OrderServiceClient
	marketData     domain.MarketDataProvider
	volumeProvider domain.VolumeProfileProvider
	venueRepo      domain.VenueRepository
}

// NewExecutionCommandService 构造函数。
func NewExecutionCommandService(
	tradeRepo domain.TradeRepository,
	algoRepo domain.AlgoOrderRepository,
	redisRepo domain.AlgoRedisRepository,
	eventStore domain.EventStore,
	publisher messagequeue.EventPublisher,
	orderClient orderv1.OrderServiceClient,
	marketData domain.MarketDataProvider,
	volumeProvider domain.VolumeProfileProvider,
	venueRepo domain.VenueRepository,
) *ExecutionCommandService {
	return &ExecutionCommandService{
		tradeRepo:      tradeRepo,
		algoRepo:       algoRepo,
		redisRepo:      redisRepo,
		eventStore:     eventStore,
		publisher:      publisher,
		orderClient:    orderClient,
		marketData:     marketData,
		volumeProvider: volumeProvider,
		venueRepo:      venueRepo,
	}
}

// ExecuteOrder 模拟单笔成交执行
func (s *ExecutionCommandService) ExecuteOrder(ctx context.Context, cmd ExecuteOrderCommand) (*ExecutionDTO, error) {
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

	err := s.tradeRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.tradeRepo.Save(txCtx, trade); err != nil {
			return err
		}

		// 保存领域事件
		if err := s.eventStore.Save(txCtx, trade.TradeID, trade.GetUncommittedEvents(), trade.Version()); err != nil {
			return err
		}
		trade.MarkCommitted()

		if s.publisher == nil {
			return nil
		}
		// 发布集成事件 (Outbox Pattern)
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.TradeExecutedEventType, trade.TradeID, map[string]any{
			"trade_id": trade.TradeID,
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
		ExecutionID: trade.TradeID,
		OrderID:     trade.OrderID,
		Symbol:      trade.Symbol,
		Status:      "FILLED",
		ExecutedQty: trade.ExecutedQuantity.String(),
		ExecutedPx:  trade.ExecutedPrice.String(),
		Timestamp:   trade.ExecutedAt.Unix(),
	}, nil
}

// SubmitAlgoOrder 提交算法订单
func (s *ExecutionCommandService) SubmitAlgoOrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
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

	err := s.algoRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.algoRepo.Save(txCtx, algoOrder); err != nil {
			return err
		}

		// 保存领域事件
		if err := s.eventStore.Save(txCtx, algoOrder.AlgoID, algoOrder.GetUncommittedEvents(), algoOrder.Version()); err != nil {
			return err
		}
		algoOrder.MarkCommitted()

		// 缓存实时状态
		if err := s.redisRepo.Save(txCtx, algoOrder); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.AlgoOrderStartedEventType, algoOrder.AlgoID, map[string]any{
			"algo_id":   algoOrder.AlgoID,
			"user_id":   algoOrder.UserID,
			"symbol":    algoOrder.Symbol,
			"algo_type": string(algoOrder.AlgoType),
		})
	})

	if err != nil {
		return "", err
	}

	return algoID, nil
}

// SubmitSOROrder 提交智能路由订单
func (s *ExecutionCommandService) SubmitSOROrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
	algoID := fmt.Sprintf("SOR-%d", idgen.GenID())
	order := domain.NewAlgoOrder(algoID, cmd.UserID, cmd.Symbol, domain.TradeSide(cmd.Side), cmd.TotalQty, domain.AlgoTypeSOR, time.Now(), time.Now().Add(time.Hour), cmd.Params)

	venues, err := s.venueRepo.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list venues: %w", err)
	}

	strategy := &domain.SORStrategy{
		Provider: s.marketData,
		Venues:   venues,
	}
	slices, err := strategy.GenerateSlices(order)
	if err != nil {
		return "", err
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

// SubmitFIXOrder 处理 FIX 网关订单
func (s *ExecutionCommandService) SubmitFIXOrder(ctx context.Context, cmd SubmitFIXOrderCommand) (*ExecutionDTO, error) {
	executeCmd := ExecuteOrderCommand{
		OrderID:  cmd.ClOrdID,
		UserID:   cmd.UserID,
		Symbol:   cmd.Symbol,
		Side:     cmd.Side,
		Price:    cmd.Price,
		Quantity: cmd.Quantity,
	}
	return s.ExecuteOrder(ctx, executeCmd)
}

// StartAlgoWorker 运行后台算法执行线程
func (s *ExecutionCommandService) StartAlgoWorker(ctx context.Context) {
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

func (s *ExecutionCommandService) processActiveAlgoOrders(ctx context.Context) {
	orders, err := s.algoRepo.ListActive(ctx)
	if err != nil {
		return
	}

	for _, order := range orders {
		var strategy domain.ExecutionStrategy
		switch order.AlgoType {
		case domain.AlgoTypeTWAP:
			strategy = &domain.TWAPStrategy{
				MinSliceSize: order.TotalQuantity.Div(decimal.NewFromInt(10)),
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
			venues, err := s.venueRepo.List(ctx)
			if err != nil {
				continue
			}
			strategy = &domain.SORStrategy{
				Provider: s.marketData,
				Venues:   venues,
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

			_, _ = s.orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
				UserId:   order.UserID,
				Symbol:   order.Symbol,
				Side:     side,
				Type:     otype,
				Price:    slice.Price.InexactFloat64(),
				Quantity: slice.Quantity.InexactFloat64(),
			})

			// 通过领域事件更新已成交量
			order.ApplyChange(&domain.TradeExecutedEvent{
				TradeID:  fmt.Sprintf("TRD-ALGO-%d", idgen.GenID()),
				OrderID:  "ALGO-CHILD",
				UserID:   order.UserID,
				Symbol:   order.Symbol,
				Quantity: slice.Quantity.String(),
				Price:    slice.Price.String(),
				Time:     time.Now().Unix(),
			})
		}

		_ = s.algoRepo.WithTx(ctx, func(txCtx context.Context) error {
			if err := s.algoRepo.Save(txCtx, order); err != nil {
				return err
			}
			if err := s.eventStore.Save(txCtx, order.AlgoID, order.GetUncommittedEvents(), order.Version()); err != nil {
				return err
			}
			order.MarkCommitted()
			return s.redisRepo.Save(txCtx, order)
		})
	}
}

// HandleLiquidation 处理强平触发事件，执行市价成交。
func (s *ExecutionCommandService) HandleLiquidation(ctx context.Context, userID, symbol, side string, qty float64) error {
	var orderSide orderv1.OrderSide
	// 强平需要执行反向操作
	if side == "buy" || side == "BUY" {
		orderSide = orderv1.OrderSide_SELL
	} else {
		orderSide = orderv1.OrderSide_BUY
	}

	_, err := s.orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:   userID,
		Symbol:   symbol,
		Side:     orderSide,
		Type:     orderv1.OrderType_MARKET,
		Quantity: qty,
	})
	if err != nil {
		return fmt.Errorf("failed to create liquidation market order: %w", err)
	}

	return nil
}
