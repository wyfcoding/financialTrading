package application

import (
	"context"

	"github.com/shopspring/decimal"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/metrics"
	"gorm.io/gorm"
)

// ExecutionService 作为执行服务操作的门面。
type ExecutionService struct {
	Command *ExecutionCommandService
	Query   *ExecutionQueryService
}

// NewExecutionService 构造函数。
func NewExecutionService(
	tradeRepo domain.TradeRepository,
	searchRepo domain.TradeSearchRepository,
	algoRepo domain.AlgoOrderRepository,
	redisRepo domain.AlgoRedisRepository,
	publisher domain.EventPublisher,
	orderClient orderv1.OrderServiceClient,
	marketData domain.MarketDataProvider,
	volumeProvider domain.VolumeProfileProvider,
	metrics *metrics.Metrics,
	db *gorm.DB,
) *ExecutionService {
	return &ExecutionService{
		Command: NewExecutionCommandService(tradeRepo, algoRepo, redisRepo, publisher, orderClient, marketData, volumeProvider, metrics, db),
		Query:   NewExecutionQueryService(tradeRepo, searchRepo),
	}
}

// --- 写操作 (Delegates to Command) ---

func (s *ExecutionService) ExecuteOrder(ctx context.Context, cmd ExecuteOrderCommand) (*ExecutionDTO, error) {
	return s.Command.ExecuteOrder(ctx, cmd)
}

func (s *ExecutionService) SubmitAlgoOrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
	return s.Command.SubmitAlgoOrder(ctx, cmd)
}

func (s *ExecutionService) SubmitSOROrder(ctx context.Context, cmd SubmitAlgoCommand) (string, error) {
	return s.Command.SubmitSOROrder(ctx, cmd)
}

func (s *ExecutionService) SubmitFIXOrder(ctx context.Context, cmd SubmitFIXOrderCommand) (*ExecutionDTO, error) {
	return s.Command.SubmitFIXOrder(ctx, cmd)
}

func (s *ExecutionService) StartAlgoWorker(ctx context.Context) {
	s.Command.StartAlgoWorker(ctx)
}

// --- 读操作 (Delegates to Query) ---

func (s *ExecutionService) GetExecutionHistory(ctx context.Context, orderID string) ([]*ExecutionDTO, error) {
	return s.Query.GetExecutionHistory(ctx, orderID)
}

func (s *ExecutionService) ListExecutions(ctx context.Context, userID string) ([]*ExecutionDTO, error) {
	// 默认限额，协议未提供则使用默认值
	dtos, _, err := s.Query.ListExecutions(ctx, userID, "", 100, 0)
	return dtos, err
}

// --- DTO Definitions ---

// ExecutionDTO 执行记录 DTO
type ExecutionDTO struct {
	ExecutionID string `json:"execution_id"`
	OrderID     string `json:"order_id"`
	Symbol      string `json:"symbol,omitempty"`
	Status      string `json:"status"`
	ExecutedQty string `json:"executed_qty"`
	ExecutedPx  string `json:"executed_px"`
	Timestamp   int64  `json:"timestamp"`
}

// ExecuteOrderCommand 执行订单命令
type ExecuteOrderCommand struct {
	OrderID  string          `json:"order_id" binding:"required"`
	UserID   string          `json:"user_id" binding:"required"`
	Symbol   string          `json:"symbol" binding:"required"`
	Side     string          `json:"side" binding:"required"`
	Price    decimal.Decimal `json:"price" binding:"required"`
	Quantity decimal.Decimal `json:"quantity" binding:"required"`
}

// SubmitAlgoCommand 提交算法订单命令
type SubmitAlgoCommand struct {
	UserID    string          `json:"user_id" binding:"required"`
	Symbol    string          `json:"symbol" binding:"required"`
	Side      string          `json:"side" binding:"required"`
	TotalQty  decimal.Decimal `json:"total_qty" binding:"required"`
	AlgoType  string          `json:"algo_type" binding:"required"` // TWAP, VWAP, POV, SOR
	StartTime int64           `json:"start_time"`
	EndTime   int64           `json:"end_time"`
	Params    string          `json:"params"`
}

// SubmitFIXOrderCommand FIX 订单命令
type SubmitFIXOrderCommand struct {
	ClOrdID  string          `json:"cl_ord_id" binding:"required"`
	UserID   string          `json:"user_id" binding:"required"`
	Symbol   string          `json:"symbol" binding:"required"`
	Side     string          `json:"side" binding:"required"`
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}
