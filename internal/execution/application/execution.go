package application

import (
	"context"

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
	algoRepo domain.AlgoOrderRepository,
	publisher domain.EventPublisher,
	orderClient orderv1.OrderServiceClient,
	marketData domain.MarketDataProvider,
	volumeProvider domain.VolumeProfileProvider,
	metrics *metrics.Metrics,
	db *gorm.DB,
) *ExecutionService {
	return &ExecutionService{
		Command: NewExecutionCommandService(tradeRepo, algoRepo, publisher, orderClient, marketData, volumeProvider, metrics, db),
		Query:   NewExecutionQueryService(tradeRepo),
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
	return s.Query.ListExecutions(ctx, userID)
}
