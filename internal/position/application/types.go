package application

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// UpdatePositionCommand 更新头寸命令
type UpdatePositionCommand struct {
	UserID   string
	Symbol   string
	Side     string
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

// ChangeCostMethodCommand 变更成本计算方法命令
type ChangeCostMethodCommand struct {
	UserID string
	Symbol string
	Method string
}

// PositionDTO 持仓 DTO
type PositionDTO struct {
	PositionID        string `json:"position_id"`
	UserID            string `json:"user_id"`
	Symbol            string `json:"symbol"`
	Side              string `json:"side"`
	Quantity          string `json:"quantity"`
	EntryPrice        string `json:"entry_price"`
	CurrentPrice      string `json:"current_price,omitempty"`
	UnrealizedPnL     string `json:"unrealized_pnl,omitempty"`
	RealizedPnL       string `json:"realized_pnl"`
	MarginRequirement string `json:"margin_requirement"`
	OpenedAt          int64  `json:"opened_at"`
	ClosedAt          *int64 `json:"closed_at,omitempty"`
	Status            string `json:"status"`
}

func toPositionDTO(p *domain.Position) *PositionDTO {
	if p == nil {
		return nil
	}
	side := "buy"
	if p.Quantity.IsNegative() {
		side = "sell"
	}

	status := "OPEN"
	if p.Quantity.IsZero() {
		status = "CLOSED"
	}

	openedAt := int64(0)
	if !p.CreatedAt.IsZero() {
		openedAt = p.CreatedAt.Unix()
	}

	return &PositionDTO{
		PositionID:        fmt.Sprintf("%d", p.ID),
		UserID:            p.UserID,
		Symbol:            p.Symbol,
		Side:              side,
		Quantity:          p.Quantity.String(),
		EntryPrice:        p.AverageEntryPrice.String(),
		RealizedPnL:       p.RealizedPnL.String(),
		UnrealizedPnL:     p.UnrealizedPnL.String(),
		MarginRequirement: p.MarginRequirement.String(),
		OpenedAt:          openedAt,
		Status:            status,
	}
}

func toPositionDTOs(positions []*domain.Position) []*PositionDTO {
	if len(positions) == 0 {
		return nil
	}
	dtos := make([]*PositionDTO, 0, len(positions))
	for _, p := range positions {
		dtos = append(dtos, toPositionDTO(p))
	}
	return dtos
}
