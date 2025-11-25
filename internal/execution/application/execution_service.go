package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialTrading/internal/execution/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/utils"
	"github.com/shopspring/decimal"
)

// ExecuteOrderRequest 执行订单请求 DTO
type ExecuteOrderRequest struct {
	OrderID  string
	UserID   string
	Symbol   string
	Side     string
	Price    string
	Quantity string
}

// ExecutionDTO 执行记录 DTO
type ExecutionDTO struct {
	ExecutionID      string
	OrderID          string
	UserID           string
	Symbol           string
	Side             string
	ExecutedPrice    string
	ExecutedQuantity string
	Status           string
	CreatedAt        int64
	UpdatedAt        int64
}

// ExecutionApplicationService 执行应用服务
type ExecutionApplicationService struct {
	executionRepo domain.ExecutionRepository
	snowflake     *utils.SnowflakeID
}

// NewExecutionApplicationService 创建执行应用服务
func NewExecutionApplicationService(executionRepo domain.ExecutionRepository) *ExecutionApplicationService {
	return &ExecutionApplicationService{
		executionRepo: executionRepo,
		snowflake:     utils.NewSnowflakeID(2),
	}
}

// ExecuteOrder 执行订单
// 用例流程：
// 1. 验证订单参数
// 2. 生成执行 ID
// 3. 创建执行记录
// 4. 保存到仓储
// 5. 发布执行事件（待实现）
func (eas *ExecutionApplicationService) ExecuteOrder(ctx context.Context, req *ExecuteOrderRequest) (*ExecutionDTO, error) {
	// 验证输入
	if req.OrderID == "" || req.UserID == "" || req.Symbol == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 解析价格和数量
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 生成执行 ID
	executionID := fmt.Sprintf("EXEC-%d", eas.snowflake.Generate())

	// 创建执行记录
	execution := &domain.Execution{
		ExecutionID:      executionID,
		OrderID:          req.OrderID,
		UserID:           req.UserID,
		Symbol:           req.Symbol,
		Side:             req.Side,
		ExecutedPrice:    price,
		ExecutedQuantity: quantity,
		Status:           domain.ExecutionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 保存到仓储
	if err := eas.executionRepo.Save(ctx, execution); err != nil {
		logger.Error(ctx, "Failed to save execution",
			"execution_id", executionID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	logger.Debug(ctx, "Order executed successfully",
		"execution_id", executionID,
		"order_id", req.OrderID,
	)

	// 转换为 DTO
	return &ExecutionDTO{
		ExecutionID:      execution.ExecutionID,
		OrderID:          execution.OrderID,
		UserID:           execution.UserID,
		Symbol:           execution.Symbol,
		Side:             execution.Side,
		ExecutedPrice:    execution.ExecutedPrice.String(),
		ExecutedQuantity: execution.ExecutedQuantity.String(),
		Status:           string(execution.Status),
		CreatedAt:        execution.CreatedAt.Unix(),
		UpdatedAt:        execution.UpdatedAt.Unix(),
	}, nil
}

// GetExecutionHistory 获取执行历史
func (eas *ExecutionApplicationService) GetExecutionHistory(ctx context.Context, userID string, limit, offset int) ([]*ExecutionDTO, int64, error) {
	// 验证输入
	if userID == "" {
		return nil, 0, fmt.Errorf("user_id is required")
	}

	// 获取执行历史
	executions, total, err := eas.executionRepo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		logger.Error(ctx, "Failed to get execution history",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get execution history: %w", err)
	}

	// 转换为 DTO 列表
	dtos := make([]*ExecutionDTO, 0, len(executions))
	for _, execution := range executions {
		dtos = append(dtos, &ExecutionDTO{
			ExecutionID:      execution.ExecutionID,
			OrderID:          execution.OrderID,
			UserID:           execution.UserID,
			Symbol:           execution.Symbol,
			Side:             execution.Side,
			ExecutedPrice:    execution.ExecutedPrice.String(),
			ExecutedQuantity: execution.ExecutedQuantity.String(),
			Status:           string(execution.Status),
			CreatedAt:        execution.CreatedAt.Unix(),
			UpdatedAt:        execution.UpdatedAt.Unix(),
		})
	}

	return dtos, total, nil
}
