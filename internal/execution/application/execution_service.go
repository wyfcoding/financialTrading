// 包 执行服务的用例逻辑 (Use Cases)。
// 这一层负责编排领域对象、仓储和外部服务（如果需要），以完成具体的业务功能。
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// ExecuteOrderRequest 是执行订单请求的数据传输对象 (DTO)。
// 它用于从接口层（如 gRPC 或 HTTP handler）向应用层传递参数，实现了与内部领域模型的解耦。
type ExecuteOrderRequest struct {
	OrderID  string // 订单 ID，由客户端或上游服务提供
	UserID   string // 用户 ID
	Symbol   string // 交易对符号，例如 "BTC/USDT"
	Side     string // 买卖方向, "BUY" 或 "SELL"
	Price    string // 订单价格 (使用字符串以保持高精度)
	Quantity string // 订单数量 (使用字符串以保持高精度)
}

// ExecutionDTO 是执行记录的数据传输对象 (DTO)。
// 用于从应用层向接口层返回执行结果，避免直接暴露领域模型。
type ExecutionDTO struct {
	ExecutionID      string `json:"execution_id"`      // 执行记录 ID
	OrderID          string `json:"order_id"`          // 关联的订单 ID
	UserID           string `json:"user_id"`           // 用户 ID
	Symbol           string `json:"symbol"`            // 交易对符号
	Side             string `json:"side"`              // 买卖方向
	ExecutedPrice    string `json:"executed_price"`    // 成交价格
	ExecutedQuantity string `json:"executed_quantity"` // 成交数量
	Status           string `json:"status"`            // 执行状态
	CreatedAt        int64  `json:"created_at"`        // 创建时间戳 (Unix seconds)
	UpdatedAt        int64  `json:"updated_at"`        // 更新时间戳 (Unix seconds)
}

// ExecutionApplicationService 是执行应用服务。
// 它封装了所有与订单执行相关的业务用例。
type ExecutionApplicationService struct {
	executionRepo domain.ExecutionRepository // 依赖注入的执行仓储接口
}

// NewExecutionApplicationService 是 ExecutionApplicationService 的构造函数。
func NewExecutionApplicationService(executionRepo domain.ExecutionRepository) *ExecutionApplicationService {
	// 初始化雪花ID生成器，传入一个唯一的节点ID（此处为2）。
	return &ExecutionApplicationService{
		executionRepo: executionRepo,
	}
}

// ExecuteOrder 是执行一个订单的业务用例。
// 简化流程:
// 1. 验证输入参数的有效性。
// 2. 将字符串格式的价格和数量转换为高精度的 decimal 类型。
// 3. 生成一个全局唯一的执行ID。
// 4. 创建一个 `Execution` 领域实体。
// 5. 通过仓储接口将该实体持久化。
// 6. 将持久化后的实体转换为 DTO 并返回。
//
// 实际场景中可能更复杂，会包括：
// - 与撮合引擎的交互。
// - 订单状态的复杂管理（如部分成交）。
// - 发布订单执行事件到消息队列（如 Kafka），供其他服务（如清算、通知）消费。
func (eas *ExecutionApplicationService) ExecuteOrder(ctx context.Context, req *ExecuteOrderRequest) (*ExecutionDTO, error) {
	// 记录性能监控
	defer logging.LogDuration(ctx, "Order execution completed",
		"order_id", req.OrderID,
		"symbol", req.Symbol,
	)()

	logging.Info(ctx, "Executing order",
		"order_id", req.OrderID,
		"user_id", req.UserID,
		"symbol", req.Symbol,
		"side", req.Side,
	)

	// 1. 输入校验
	if req.OrderID == "" || req.UserID == "" || req.Symbol == "" {
		logging.Warn(ctx, "Invalid execution request parameters",
			"order_id", req.OrderID,
			"user_id", req.UserID,
			"symbol", req.Symbol,
		)
		return nil, fmt.Errorf("invalid request parameters: OrderID, UserID, and Symbol are required")
	}

	// 2. 数据转换和校验
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		logging.Error(ctx, "Failed to parse execution price",
			"order_id", req.OrderID,
			"price", req.Price,
			"error", err,
		)
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		logging.Error(ctx, "Failed to parse execution quantity",
			"order_id", req.OrderID,
			"quantity", req.Quantity,
			"error", err,
		)
		return nil, fmt.Errorf("invalid quantity format: %w", err)
	}

	// 3. 生成唯一ID
	executionID := fmt.Sprintf("EXEC-%d", idgen.GenID())

	logging.Debug(ctx, "Generated execution ID",
		"execution_id", executionID,
		"order_id", req.OrderID,
	)

	// 4. 创建领域实体
	execution := &domain.Execution{
		ExecutionID:      executionID,
		OrderID:          req.OrderID,
		UserID:           req.UserID,
		Symbol:           req.Symbol,
		Side:             domain.OrderSide(req.Side),
		ExecutedPrice:    price,
		ExecutedQuantity: quantity,
		Status:           domain.ExecutionStatusCompleted, // 简化处理，假设订单立即完全成交
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 5. 持久化
	if err := eas.executionRepo.Save(ctx, execution); err != nil {
		logging.Error(ctx, "Failed to save execution record",
			"execution_id", executionID,
			"order_id", req.OrderID,
			"user_id", req.UserID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save execution record: %w", err)
	}

	logging.Info(ctx, "Order executed successfully",
		"execution_id", executionID,
		"order_id", req.OrderID,
		"user_id", req.UserID,
		"symbol", req.Symbol,
		"price", price.String(),
		"quantity", quantity.String(),
	)

	// 6. 转换为 DTO 并返回
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

// GetExecutionHistory 是获取指定用户交易历史的业务用例。
func (eas *ExecutionApplicationService) GetExecutionHistory(ctx context.Context, userID string, limit, offset int) ([]*ExecutionDTO, int64, error) {
	if userID == "" {
		return nil, 0, fmt.Errorf("user_id is required")
	}

	// 从仓储获取领域实体列表
	executions, total, err := eas.executionRepo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		logging.Error(ctx, "Failed to get execution history",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get execution history: %w", err)
	}

	// 将领域实体列表转换为 DTO 列表
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
