package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// PositionQueryService 处理所有持仓相关的查询操作（Queries）。
type PositionQueryService struct {
	repo domain.PositionRepository
}

// NewPositionQueryService 构造函数。
func NewPositionQueryService(repo domain.PositionRepository) *PositionQueryService {
	return &PositionQueryService{repo: repo}
}

func (s *PositionQueryService) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	positions, total, err := s.repo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*PositionDTO, 0, len(positions))
	for _, p := range positions {
		dtos = append(dtos, s.mapToDTO(p))
	}
	return dtos, total, nil
}

func (s *PositionQueryService) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	pos, err := s.repo.Get(ctx, positionID)
	if err != nil || pos == nil {
		return nil, err
	}
	return s.mapToDTO(pos), nil
}

func (s *PositionQueryService) mapToDTO(p *domain.Position) *PositionDTO {
	side := "buy"
	if p.Quantity < 0 {
		side = "sell"
	}
	return &PositionDTO{
		PositionID:  fmt.Sprintf("%d", p.ID),
		UserID:      p.UserID,
		Symbol:      p.Symbol,
		Side:        side,
		Quantity:    fmt.Sprintf("%f", p.Quantity),
		EntryPrice:  fmt.Sprintf("%f", p.AverageEntryPrice),
		RealizedPnL: fmt.Sprintf("%f", p.RealizedPnL),
		OpenedAt:    p.CreatedAt.Unix(),
		Status:      "OPEN",
	}
}
