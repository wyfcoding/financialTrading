package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// PositionQueryService 处理所有持仓相关的查询操作（Queries）。
type PositionQueryService struct {
	repo     domain.PositionRepository
	readRepo domain.PositionReadRepository
}

// NewPositionQueryService 构造函数。
func NewPositionQueryService(repo domain.PositionRepository, readRepo domain.PositionReadRepository) *PositionQueryService {
	return &PositionQueryService{repo: repo, readRepo: readRepo}
}

func (s *PositionQueryService) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	positions, total, err := s.repo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toPositionDTOs(positions), total, nil
}

func (s *PositionQueryService) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.Get(ctx, positionID); err == nil && cached != nil {
			return toPositionDTO(cached), nil
		}
	}

	pos, err := s.repo.Get(ctx, positionID)
	if err != nil || pos == nil {
		return nil, err
	}

	if s.readRepo != nil {
		_ = s.readRepo.Save(ctx, pos)
	}

	return toPositionDTO(pos), nil
}
