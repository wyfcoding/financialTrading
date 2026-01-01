package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// PositionQuery 处理所有持仓相关的查询操作（Queries）。
type PositionQuery struct {
	repo domain.PositionRepository
}

// NewPositionQuery 构造函数。
func NewPositionQuery(repo domain.PositionRepository) *PositionQuery {
	return &PositionQuery{repo: repo}
}

// GetPositions 获取用户持仓列表
func (q *PositionQuery) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	if userID == "" {
		return nil, 0, fmt.Errorf("user_id is required")
	}

	positions, total, err := q.repo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*PositionDTO, 0, len(positions))
	for _, position := range positions {
		dtos = append(dtos, mapToPositionDTO(position))
	}

	return dtos, total, nil
}

// GetPosition 获取持仓详情
func (q *PositionQuery) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	position, err := q.repo.Get(ctx, positionID)
	if err != nil {
		return nil, err
	}
	if position == nil {
		return nil, fmt.Errorf("position not found")
	}

	return mapToPositionDTO(position), nil
}

func mapToPositionDTO(position *domain.Position) *PositionDTO {
	var closedAt *int64
	if position.ClosedAt != nil {
		ts := position.ClosedAt.Unix()
		closedAt = &ts
	}

	return &PositionDTO{
		PositionID:    position.PositionID,
		UserID:        position.UserID,
		Symbol:        position.Symbol,
		Side:          position.Side,
		Quantity:      position.Quantity.String(),
		EntryPrice:    position.EntryPrice.String(),
		CurrentPrice:  position.CurrentPrice.String(),
		UnrealizedPnL: position.UnrealizedPnL.String(),
		RealizedPnL:   position.RealizedPnL.String(),
		OpenedAt:      position.OpenedAt.Unix(),
		ClosedAt:      closedAt,
		Status:        position.Status,
	}
}
