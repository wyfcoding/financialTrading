package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// PositionService 持仓门面服务。
type PositionService struct {
	Command *PositionCommandService
	Query   *PositionQueryService
	logger  *slog.Logger
}

// NewPositionService 构造函数。
func NewPositionService(repo domain.PositionRepository, logger *slog.Logger) *PositionService {
	return &PositionService{
		Command: NewPositionCommandService(repo, logger),
		Query:   NewPositionQueryService(repo),
		logger:  logger.With("module", "position_service"),
	}
}

func (s *PositionService) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	return s.Command.ClosePosition(ctx, positionID, closePrice)
}

// --- TCC Facade ---

func (s *PositionService) TccTryFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.Command.TccTryFreeze(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) TccConfirmFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.Command.TccConfirmFreeze(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) TccCancelFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.Command.TccCancelFreeze(ctx, barrier, userID, symbol, quantity)
}

// --- Saga Facade ---

func (s *PositionService) SagaDeductFrozen(ctx context.Context, barrier any, userID, symbol string, quantity, price decimal.Decimal) error {
	return s.Command.SagaDeductFrozen(ctx, barrier, userID, symbol, quantity, price)
}

func (s *PositionService) SagaRefundFrozen(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.Command.SagaRefundFrozen(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) SagaAddPosition(ctx context.Context, barrier any, userID, symbol string, quantity, price decimal.Decimal) error {
	return s.Command.SagaAddPosition(ctx, barrier, userID, symbol, quantity, price)
}

func (s *PositionService) SagaSubPosition(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.Command.SagaSubPosition(ctx, barrier, userID, symbol, quantity)
}

// --- Query (Reads) ---

func (s *PositionService) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	return s.Query.GetPositions(ctx, userID, limit, offset)
}

func (s *PositionService) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	return s.Query.GetPosition(ctx, positionID)
}
