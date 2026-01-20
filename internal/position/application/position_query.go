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
	// Adapt new domain model to DTO
	// Position now uses float64 and implicit side via quantity sign (if applicable, or just qty > 0)
	// Missing fields: ClosedAt, Side, CurrentPrice (removed from domain).
	// We infer Side.

	side := "buy" // Default long
	if position.Quantity < 0 {
		side = "sell" // short
	}

	// Status? Position struct doesn't have it. Assume "OPEN" if returned by Repo.
	status := "OPEN"

	return &PositionDTO{
		PositionID:    fmt.Sprintf("%d", position.ID), // ID is uint in gorm.Model
		UserID:        position.UserID,
		Symbol:        position.Symbol,
		Side:          side,
		Quantity:      fmt.Sprintf("%f", position.Quantity),
		EntryPrice:    fmt.Sprintf("%f", position.AverageEntryPrice),
		CurrentPrice:  "0", // Not available in domain
		UnrealizedPnL: "0", // Not available in domain (calculated dynamic)
		RealizedPnL:   fmt.Sprintf("%f", position.RealizedPnL),
		OpenedAt:      position.CreatedAt.Unix(),
		ClosedAt:      nil,
		Status:        status,
	}
}
