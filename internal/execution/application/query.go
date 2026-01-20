package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type ExecutionQueryService struct {
	tradeRepo domain.TradeRepository
}

func NewExecutionQueryService(tradeRepo domain.TradeRepository) *ExecutionQueryService {
	return &ExecutionQueryService{tradeRepo: tradeRepo}
}

func (q *ExecutionQueryService) GetExecutionHistory(ctx context.Context, orderID string) ([]*ExecutionDTO, error) {
	trade, err := q.tradeRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if trade == nil {
		return []*ExecutionDTO{}, nil
	}

	// Return single trade in slice
	return []*ExecutionDTO{{
		ExecutionID: trade.ID,
		OrderID:     trade.OrderID,
		Status:      "FILLED",
		ExecutedQty: trade.ExecutedQuantity.String(),
		ExecutedPx:  trade.ExecutedPrice.String(),
		Timestamp:   trade.ExecutedAt.Unix(),
	}}, nil
}
