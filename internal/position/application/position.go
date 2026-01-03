package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// PositionService 持仓门面服务，整合 Manager 和 Query。
type PositionService struct {
	manager *PositionManager
	query   *PositionQuery
}

// NewPositionService 构造函数。
func NewPositionService(repo domain.PositionRepository) *PositionService {
	return &PositionService{
		manager: NewPositionManager(repo),
		query:   NewPositionQuery(repo),
	}
}

// --- Manager (Writes) ---

func (s *PositionService) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	return s.manager.ClosePosition(ctx, positionID, closePrice)
}

// --- TCC Facade ---

func (s *PositionService) TccTryFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return s.manager.TccTryFreeze(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return s.manager.TccConfirmFreeze(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return s.manager.TccCancelFreeze(ctx, barrier, userID, symbol, quantity)
}

// --- Saga Facade ---

func (s *PositionService) SagaDeductFrozen(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return s.manager.SagaDeductFrozen(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) SagaRefundFrozen(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return s.manager.SagaRefundFrozen(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionService) SagaAddPosition(ctx context.Context, barrier interface{}, userID, symbol string, quantity, price decimal.Decimal) error {
	return s.manager.SagaAddPosition(ctx, barrier, userID, symbol, quantity, price)
}

func (s *PositionService) SagaSubPosition(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return s.manager.SagaSubPosition(ctx, barrier, userID, symbol, quantity)
}

// --- Query (Reads) ---

func (s *PositionService) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	return s.query.GetPositions(ctx, userID, limit, offset)
}

func (s *PositionService) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	return s.query.GetPosition(ctx, positionID)
}

// --- Legacy Compatibility Types ---

// PositionDTO 持仓 DTO
type PositionDTO struct {
	PositionID    string
	UserID        string
	Symbol        string
	Side          string
	Quantity      string
	EntryPrice    string
	CurrentPrice  string
	UnrealizedPnL string
	RealizedPnL   string
	OpenedAt      int64
	ClosedAt      *int64
	Status        string
}
