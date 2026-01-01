package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// ExecutionManager 处理所有订单执行相关的写入操作（Commands）。
type ExecutionManager struct {
	repo domain.ExecutionRepository
}

// NewExecutionManager 构造函数。
func NewExecutionManager(repo domain.ExecutionRepository) *ExecutionManager {
	return &ExecutionManager{repo: repo}
}

// ExecuteOrder 执行订单
func (m *ExecutionManager) ExecuteOrder(ctx context.Context, req *ExecuteOrderRequest) (*ExecutionDTO, error) {
	// 记录性能监控
	defer logging.LogDuration(ctx, "Order execution completed",
		"order_id", req.OrderID,
		"symbol", req.Symbol,
	)()

	logging.Info(ctx, "Executing order",
		"order_id", req.OrderID,
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)

	// 1. 输入校验
	if req.OrderID == "" || req.UserID == "" || req.Symbol == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 2. 数据转换和校验
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity format: %w", err)
	}

	// 3. 生成唯一ID
	executionID := fmt.Sprintf("EXEC-%d", idgen.GenID())

	// 4. 创建领域实体
	execution := &domain.Execution{
		ExecutionID:      executionID,
		OrderID:          req.OrderID,
		UserID:           req.UserID,
		Symbol:           req.Symbol,
		Side:             domain.OrderSide(req.Side),
		ExecutedPrice:    price,
		ExecutedQuantity: quantity,
		Status:           domain.ExecutionStatusCompleted,
	}

	// 5. 持久化
	if err := m.repo.Save(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution record: %w", err)
	}

	return &ExecutionDTO{
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
	}, nil
}
