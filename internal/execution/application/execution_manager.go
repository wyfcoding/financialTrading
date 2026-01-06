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

// ExecuteOrder 执行常规的即时成交订单指令。
func (m *ExecutionManager) ExecuteOrder(ctx context.Context, req *ExecuteOrderRequest) (*ExecutionDTO, error) {
	// 记录操作性能
	defer logging.LogDuration(ctx, "order execution completed",
		"order_id", req.OrderID,
		"symbol", req.Symbol,
	)()

	logging.Info(ctx, "executing order",
		"order_id", req.OrderID,
		"user_id", req.UserID,
		"symbol", req.Symbol,
	)

	// ... (参数校验与实体创建保持不变) ...
	if req.OrderID == "" || req.UserID == "" || req.Symbol == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity format: %w", err)
	}

	executionID := fmt.Sprintf("EXEC-%d", idgen.GenID())

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

	if err := m.repo.Save(ctx, execution); err != nil {
		logging.Error(ctx, "failed to save execution record", "order_id", req.OrderID, "error", err)
		return nil, fmt.Errorf("failed to save execution record: %w", err)
	}

	logging.Info(ctx, "order executed successfully", "order_id", req.OrderID, "exec_id", executionID)
	// ... (DTO 组装) ...
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

// SubmitAlgoOrder 提交复杂的算法执行指令（如 VWAP/TWAP），启动异步执行引擎。
// 架构设计：保存指令状态 -> 触发算法引擎协程 -> 返回跟踪 ID。
func (m *ExecutionManager) SubmitAlgoOrder(ctx context.Context, req *executionv1.SubmitAlgoOrderRequest) (*executionv1.SubmitAlgoOrderResponse, error) {
	qty, err := decimal.NewFromString(req.TotalQuantity)
	if err != nil {
		return nil, fmt.Errorf("invalid total quantity: %w", err)
	}

	rate := decimal.Zero
	if req.ParticipationRate != "" {
		parsedRate, err := decimal.NewFromString(req.ParticipationRate)
		if err != nil {
			return nil, fmt.Errorf("invalid participation rate format: %w", err)
		}
		rate = parsedRate
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
		logging.Error(ctx, "failed to persist algo order", "user_id", req.UserId, "error", err)
		return nil, fmt.Errorf("failed to save algo order: %w", err)
	}

	// 启动算法管理器中的实时监控与下单协程
	if m.algoMgr != nil {
		m.algoMgr.Start(ctx, algo)
	}

	logging.Info(ctx, "algo order submitted successfully", "algo_id", algoID, "algo_type", req.AlgoType)
	return &executionv1.SubmitAlgoOrderResponse{
		AlgoId: algoID,
		Status: string(algo.Status),
	}, nil
}

// SubmitSOROrder 提交智能路由执行（Smart Order Routing），自动拆分订单至流动性最优的市场。
func (m *ExecutionManager) SubmitSOROrder(ctx context.Context, req *executionv1.SubmitSOROrderRequest) (*executionv1.SubmitSOROrderResponse, error) {
	if m.sorMgr == nil {
		return nil, fmt.Errorf("SOR manager not initialized")
	}
	resp, err := m.sorMgr.ExecuteSOR(ctx, req)
	if err != nil {
		logging.Error(ctx, "sor execution failed", "user_id", req.UserId, "symbol", req.Symbol, "error", err)
		return nil, err
	}
	logging.Info(ctx, "sor order processed", "user_id", req.UserId, "symbol", req.Symbol)
	return resp, nil
}
