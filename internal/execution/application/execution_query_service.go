package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

// ExecutionQueryService 处理所有执行相关的查询操作（Queries）。
type ExecutionQueryService struct {
	tradeRepo domain.TradeRepository
}

// NewExecutionQueryService 构造函数。
func NewExecutionQueryService(tradeRepo domain.TradeRepository) *ExecutionQueryService {
	return &ExecutionQueryService{tradeRepo: tradeRepo}
}

// GetExecutionHistory 获取执行历史 (按订单 ID)
func (q *ExecutionQueryService) GetExecutionHistory(ctx context.Context, orderID string) ([]*ExecutionDTO, error) {
	trade, err := q.tradeRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if trade == nil {
		return []*ExecutionDTO{}, nil
	}

	return []*ExecutionDTO{q.toDTO(trade)}, nil
}

// ListExecutions 获取指定用户的所有执行历史
func (q *ExecutionQueryService) ListExecutions(ctx context.Context, userID string) ([]*ExecutionDTO, error) {
	trades, err := q.tradeRepo.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*ExecutionDTO, 0, len(trades))
	for _, t := range trades {
		dtos = append(dtos, q.toDTO(t))
	}
	return dtos, nil
}

func (q *ExecutionQueryService) toDTO(t *domain.Trade) *ExecutionDTO {
	return &ExecutionDTO{
		ExecutionID: t.TradeID,
		OrderID:     t.OrderID,
		Symbol:      t.Symbol,
		Status:      "FILLED",
		ExecutedQty: t.ExecutedQuantity.String(),
		ExecutedPx:  t.ExecutedPrice.String(),
		Timestamp:   t.ExecutedAt.Unix(),
	}
}
