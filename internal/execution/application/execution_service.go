// Package application 包含执行服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/execution/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/wyfcoding/financialTrading/pkg/utils"
)

// ExecuteOrderRequest 执行订单请求 DTO
// 用于接收来自上层（如 API 层）的订单执行请求参数
type ExecuteOrderRequest struct {
	OrderID  string // 订单 ID，全局唯一
	UserID   string // 用户 ID
	Symbol   string // 交易对符号，例如 "BTC/USD"
	Side     string // 买卖方向，"buy" 或 "sell"
	Price    string // 执行价格，使用字符串避免精度丢失
	Quantity string // 执行数量，使用字符串避免精度丢失
}

// ExecutionDTO 执行记录 DTO
// 用于向外层返回执行结果数据
type ExecutionDTO struct {
	ExecutionID      string // 执行记录 ID，全局唯一
	OrderID          string // 关联的订单 ID
	UserID           string // 用户 ID
	Symbol           string // 交易对符号
	Side             string // 买卖方向
	ExecutedPrice    string // 成交价格
	ExecutedQuantity string // 成交数量
	Status           string // 执行状态
	CreatedAt        int64  // 创建时间戳（秒）
	UpdatedAt        int64  // 更新时间戳（秒）
}

// ExecutionApplicationService 执行应用服务
// 负责处理订单执行的核心业务逻辑
type ExecutionApplicationService struct {
	executionRepo domain.ExecutionRepository // 执行记录仓储接口
	snowflake     *utils.SnowflakeID         // 雪花算法 ID 生成器
}

// NewExecutionApplicationService 创建执行应用服务
// executionRepo: 注入的执行记录仓储实现
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
