package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

// ExecutionQuery 处理所有订单执行相关的查询操作（Queries）。
type ExecutionQuery struct {
	repo domain.ExecutionRepository
}

// NewExecutionQuery 构造函数。
func NewExecutionQuery(repo domain.ExecutionRepository) *ExecutionQuery {
	return &ExecutionQuery{repo: repo}
}

// GetExecutionHistory 获取用户执行历史
func (q *ExecutionQuery) GetExecutionHistory(ctx context.Context, userID string, limit, offset int) ([]*ExecutionDTO, int64, error) {
	if userID == "" {
		return nil, 0, fmt.Errorf("user_id is required")
	}

	executions, total, err := q.repo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*ExecutionDTO, 0, len(executions))
	for _, execution := range executions {
		dtos = append(dtos, &ExecutionDTO{
			ExecutionID:      execution.ExecutionID,
			OrderID:          execution.OrderID,
			UserID:           execution.UserID,
			Symbol:           execution.Symbol,
			Side:             string(execution.Side),
			ExecutedPrice:    execution.ExecutedPrice.String(),
			ExecutedQuantity: execution.ExecutedQuantity.String(),
			Status:           string(execution.Status),
			CreatedAt:        execution.CreatedAt.Unix(),
			UpdatedAt:        execution.UpdatedAt.Unix(),
		})
	}

	return dtos, total, nil
}
