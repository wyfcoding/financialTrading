package application

import (
	"context"

	executionv1 "github.com/wyfcoding/financialtrading/goapi/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

// ExecutionService 执行门面服务，整合 Manager 和 Query。
type ExecutionService struct {
	manager *ExecutionManager
	query   *ExecutionQuery
}

// NewExecutionService 构造函数。
func NewExecutionService(repo domain.ExecutionRepository) *ExecutionService {
	return &ExecutionService{
		manager: NewExecutionManager(repo),
		query:   NewExecutionQuery(repo),
	}
}

// --- Manager (Writes) ---

func (s *ExecutionService) ExecuteOrder(ctx context.Context, req *ExecuteOrderRequest) (*ExecutionDTO, error) {
	return s.manager.ExecuteOrder(ctx, req)
}

func (s *ExecutionService) SubmitAlgoOrder(ctx context.Context, req *executionv1.SubmitAlgoOrderRequest) (*executionv1.SubmitAlgoOrderResponse, error) {
	return s.manager.SubmitAlgoOrder(ctx, req)
}

func (s *ExecutionService) SubmitSOROrder(ctx context.Context, req *executionv1.SubmitSOROrderRequest) (*executionv1.SubmitSOROrderResponse, error) {
	return s.manager.SubmitSOROrder(ctx, req)
}

func (s *ExecutionService) SetAlgoManager(algoMgr *AlgoManager) {
	s.manager.SetAlgoManager(algoMgr)
}

func (s *ExecutionService) SetSORManager(sorMgr *SORManager) {
	s.manager.SetSORManager(sorMgr)
}

// --- Query (Reads) ---

func (s *ExecutionService) GetExecutionHistory(ctx context.Context, userID string, limit, offset int) ([]*ExecutionDTO, int64, error) {
	return s.query.GetExecutionHistory(ctx, userID, limit, offset)
}

// --- Legacy Compatibility Types ---

// ExecuteOrderRequest 是执行订单请求 DTO
type ExecuteOrderRequest struct {
	OrderID  string // 订单 ID
	UserID   string // 用户 ID
	Symbol   string // 交易对符号
	Side     string // 买卖方向
	Price    string // 价格
	Quantity string // 数量
}

// ExecutionDTO 是执行记录 DTO
type ExecutionDTO struct {
	ExecutionID      string `json:"execution_id"`
	OrderID          string `json:"order_id"`
	UserID           string `json:"user_id"`
	Symbol           string `json:"symbol"`
	Side             string `json:"side"`
	ExecutedPrice    string `json:"executed_price"`
	ExecutedQuantity string `json:"executed_quantity"`
	Status           string `json:"status"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}
