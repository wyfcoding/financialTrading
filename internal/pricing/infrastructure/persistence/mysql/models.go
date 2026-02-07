package mysql

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PriceModel MySQL 价格表映射
type PriceModel struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	Symbol    string    `gorm:"column:symbol;type:varchar(20);index;not null"`
	Bid       float64   `gorm:"column:bid;type:decimal(20,8)"`
	Ask       float64   `gorm:"column:ask;type:decimal(20,8)"`
	Mid       float64   `gorm:"column:mid;type:decimal(20,8)"`
	Source    string    `gorm:"column:source;type:varchar(50)"`
	Timestamp time.Time `gorm:"column:timestamp;index"`
}

func (PriceModel) TableName() string { return "prices" }

// PricingResultModel 定价结果数据库模型
type PricingResultModel struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
	Symbol          string    `gorm:"column:symbol;type:varchar(32);index;not null"`
	OptionPrice     string    `gorm:"column:option_price;type:decimal(32,18);not null"`
	UnderlyingPrice string    `gorm:"column:underlying_price;type:decimal(32,18);not null"`
	Delta           string    `gorm:"column:delta;type:decimal(32,18)"`
	Gamma           string    `gorm:"column:gamma;type:decimal(32,18)"`
	Theta           string    `gorm:"column:theta;type:decimal(32,18)"`
	Vega            string    `gorm:"column:vega;type:decimal(32,18)"`
	Rho             string    `gorm:"column:rho;type:decimal(32,18)"`
	CalculatedAt    int64     `gorm:"column:calculated_at;type:bigint;not null"`
	PricingModel    string    `gorm:"column:pricing_model;type:varchar(32)"`
}

func (PricingResultModel) TableName() string { return "pricing_results" }

// mapping helpers

func toPriceModel(p *domain.Price) *PriceModel {
	if p == nil {
		return nil
	}
	return &PriceModel{
		ID:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		Symbol:    p.Symbol,
		Bid:       p.Bid,
		Ask:       p.Ask,
		Mid:       p.Mid,
		Source:    p.Source,
		Timestamp: p.Timestamp,
	}
}

func toPrice(m *PriceModel) *domain.Price {
	if m == nil {
		return nil
	}
	return &domain.Price{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Symbol:    m.Symbol,
		Bid:       m.Bid,
		Ask:       m.Ask,
		Mid:       m.Mid,
		Source:    m.Source,
		Timestamp: m.Timestamp,
	}
}

func toPricingResultModel(res *domain.PricingResult) *PricingResultModel {
	if res == nil {
		return nil
	}
	return &PricingResultModel{
		ID:              res.ID,
		CreatedAt:       res.CreatedAt,
		UpdatedAt:       res.UpdatedAt,
		Symbol:          res.Symbol,
		OptionPrice:     res.OptionPrice.String(),
		UnderlyingPrice: res.UnderlyingPrice.String(),
		Delta:           res.Delta.String(),
		Gamma:           res.Gamma.String(),
		Theta:           res.Theta.String(),
		Vega:            res.Vega.String(),
		Rho:             res.Rho.String(),
		CalculatedAt:    res.CalculatedAt,
		PricingModel:    res.PricingModel,
	}
}

func toPricingResult(m *PricingResultModel) *domain.PricingResult {
	if m == nil {
		return nil
	}
	opPrice, _ := decimal.NewFromString(m.OptionPrice)
	ulPrice, _ := decimal.NewFromString(m.UnderlyingPrice)
	delta, _ := decimal.NewFromString(m.Delta)
	gamma, _ := decimal.NewFromString(m.Gamma)
	theta, _ := decimal.NewFromString(m.Theta)
	vega, _ := decimal.NewFromString(m.Vega)
	rho, _ := decimal.NewFromString(m.Rho)

	return &domain.PricingResult{
		ID:              m.ID,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		Symbol:          m.Symbol,
		OptionPrice:     opPrice,
		UnderlyingPrice: ulPrice,
		Delta:           delta,
		Gamma:           gamma,
		Theta:           theta,
		Vega:            vega,
		Rho:             rho,
		CalculatedAt:    m.CalculatedAt,
		PricingModel:    m.PricingModel,
	}
}
