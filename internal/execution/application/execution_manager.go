package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	executionv1 "github.com/wyfcoding/financialtrading/goapi/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// ExecutionManager 处理所有订单执行相关的写入操作（Commands）。
type ExecutionManager struct {
	repo    domain.ExecutionRepository
	algoMgr *AlgoManager
	sorMgr  *SORManager
}

// NewExecutionManager 构造函数。
func NewExecutionManager(repo domain.ExecutionRepository) *ExecutionManager {
	return &ExecutionManager{repo: repo}
}

// SetAlgoManager 注入算法管理器。
func (m *ExecutionManager) SetAlgoManager(algoMgr *AlgoManager) {
	m.algoMgr = algoMgr
}

// SetSORManager 注入智能路由管理器。
func (m *ExecutionManager) SetSORManager(sorMgr *SORManager) {
	m.sorMgr = sorMgr
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
	}, nil
}

// SubmitAlgoOrder 提交算法订单。
func (m *ExecutionManager) SubmitAlgoOrder(ctx context.Context, req *executionv1.SubmitAlgoOrderRequest) (*executionv1.SubmitAlgoOrderResponse, error) {
	qty, err := decimal.NewFromString(req.TotalQuantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	rate := decimal.Zero
	if req.ParticipationRate != "" {
		rate, _ = decimal.NewFromString(req.ParticipationRate)
	}

	algoID := fmt.Sprintf("ALGO-%d", idgen.GenID())
	algo := &domain.AlgoOrder{
		AlgoID:            algoID,
		UserID:            req.UserId,
		Symbol:            req.Symbol,
		Side:              domain.OrderSide(req.Side),
		TotalQuantity:     qty,
		ExecutedQuantity:  decimal.Zero,
		AlgoType:          domain.AlgoType(req.AlgoType),
		StartTime:         time.Unix(req.StartTime, 0),
		EndTime:           time.Unix(req.EndTime, 0),
		ParticipationRate: rate,
		Status:            domain.ExecutionStatusExecuting,
	}

	if err := m.repo.SaveAlgoOrder(ctx, algo); err != nil {
		return nil, fmt.Errorf("failed to save algo order: %w", err)
	}

	if m.algoMgr != nil {
		m.algoMgr.Start(ctx, algo)
	}

	return &executionv1.SubmitAlgoOrderResponse{
		AlgoId: algoID,
		Status: string(algo.Status),
	}, nil
}

// SubmitSOROrder 提交智能路由订单。
func (m *ExecutionManager) SubmitSOROrder(ctx context.Context, req *executionv1.SubmitSOROrderRequest) (*executionv1.SubmitSOROrderResponse, error) {
	if m.sorMgr == nil {
		return nil, fmt.Errorf("SOR manager not initialized")
	}
	return m.sorMgr.ExecuteSOR(ctx, req)
}
