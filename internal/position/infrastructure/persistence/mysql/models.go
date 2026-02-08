package mysql

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"gorm.io/gorm"
)

// PositionModel MySQL 持仓表映射
type PositionModel struct {
	gorm.Model
	UserID            string             `gorm:"column:user_id;type:varchar(50);index;uniqueIndex:idx_user_symbol;not null"`
	Symbol            string             `gorm:"column:symbol;type:varchar(20);index;uniqueIndex:idx_user_symbol;not null"`
	Quantity          decimal.Decimal    `gorm:"column:quantity;type:decimal(32,18)"`
	AverageEntryPrice decimal.Decimal    `gorm:"column:average_entry_price;type:decimal(32,18)"`
	RealizedPnL       decimal.Decimal    `gorm:"column:realized_pnl;type:decimal(32,18);default:0"`
	Method            string             `gorm:"column:cost_method;type:varchar(20);default:'AVERAGE'"`
	Lots              []PositionLotModel `gorm:"foreignKey:PositionID;constraint:OnDelete:CASCADE"`
}

func (PositionModel) TableName() string { return "positions" }

// PositionLotModel MySQL 持仓批次表映射
type PositionLotModel struct {
	gorm.Model
	PositionID uint            `gorm:"column:position_id;index;not null"`
	Quantity   decimal.Decimal `gorm:"column:quantity;type:decimal(32,18)"`
	Price      decimal.Decimal `gorm:"column:price;type:decimal(32,18)"`
}

func (PositionLotModel) TableName() string { return "position_lots" }

// mapping helpers

func toPositionModel(p *domain.Position) *PositionModel {
	if p == nil {
		return nil
	}
	return &PositionModel{
		Model: gorm.Model{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		UserID:            p.UserID,
		Symbol:            p.Symbol,
		Quantity:          p.Quantity,
		AverageEntryPrice: p.AverageEntryPrice,
		RealizedPnL:       p.RealizedPnL,
		Method:            string(p.Method),
		Lots:              toPositionLotModels(p.Lots),
	}
}

func toPosition(m *PositionModel) *domain.Position {
	if m == nil {
		return nil
	}
	return &domain.Position{
		ID:                m.ID,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		UserID:            m.UserID,
		Symbol:            m.Symbol,
		Quantity:          m.Quantity,
		AverageEntryPrice: m.AverageEntryPrice,
		RealizedPnL:       m.RealizedPnL,
		Method:            domain.CostBasisMethod(m.Method),
		Lots:              toPositionLots(m.Lots),
	}
}

func toPositionLotModels(lots []domain.PositionLot) []PositionLotModel {
	if len(lots) == 0 {
		return nil
	}
	models := make([]PositionLotModel, len(lots))
	for i := range lots {
		models[i] = PositionLotModel{
			Model: gorm.Model{
				ID:        lots[i].ID,
				CreatedAt: lots[i].CreatedAt,
				UpdatedAt: lots[i].UpdatedAt,
			},
			PositionID: lots[i].PositionID,
			Quantity:   lots[i].Quantity,
			Price:      lots[i].Price,
		}
	}
	return models
}

func toPositionLots(models []PositionLotModel) []domain.PositionLot {
	if len(models) == 0 {
		return nil
	}
	lots := make([]domain.PositionLot, len(models))
	for i := range models {
		lots[i] = domain.PositionLot{
			ID:         models[i].ID,
			CreatedAt:  models[i].CreatedAt,
			UpdatedAt:  models[i].UpdatedAt,
			PositionID: models[i].PositionID,
			Quantity:   models[i].Quantity,
			Price:      models[i].Price,
		}
	}
	return lots
}
