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

func (s *PositionService) UpdatePositionPrice(ctx context.Context, positionID string, currentPrice decimal.Decimal) error {
	return s.manager.UpdatePositionPrice(ctx, positionID, currentPrice)
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
