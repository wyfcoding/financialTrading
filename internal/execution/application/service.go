package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

type ExecutionApplicationService struct {
	tradeRepo domain.TradeRepository
	algoRepo  domain.AlgoOrderRepository
	outbox    *outbox.Manager
	db        *gorm.DB
}

func NewExecutionApplicationService(
	tradeRepo domain.TradeRepository,
	algoRepo domain.AlgoOrderRepository,
	outbox *outbox.Manager,
	db *gorm.DB,
) *ExecutionApplicationService {
	return &ExecutionApplicationService{
		tradeRepo: tradeRepo,
		algoRepo:  algoRepo,
		outbox:    outbox,
		db:        db,
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
func (s *ExecutionApplicationService) SubmitSOROrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
	// 简化实现: 智能路由在演示中退化为普通聚合执行
	return s.SubmitAlgoOrder(ctx, cmd)
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
