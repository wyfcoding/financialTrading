package mysql

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"gorm.io/gorm"
)

// SymbolModel 交易对数据库模型
type SymbolModel struct {
	gorm.Model
	ID             string `gorm:"column:id;type:varchar(32);primaryKey"`
	BaseCurrency   string `gorm:"column:base_currency;type:varchar(10);not null"`
	QuoteCurrency  string `gorm:"column:quote_currency;type:varchar(10);not null"`
	ExchangeID     string `gorm:"column:exchange_id;type:varchar(32);not null;index"`
	SymbolCode     string `gorm:"column:symbol_code;type:varchar(20);uniqueIndex;not null"`
	Status         string `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
	MinOrderSize   string `gorm:"column:min_order_size;type:decimal(20,8)"`
	PricePrecision string `gorm:"column:price_precision;type:decimal(20,8)"`
}

func (SymbolModel) TableName() string {
	return "symbols"
}

func (m *SymbolModel) ToDomain() *domain.Symbol {
	minSize, _ := decimal.NewFromString(m.MinOrderSize)
	pricePrec, _ := decimal.NewFromString(m.PricePrecision)
	return &domain.Symbol{
		Model:          m.Model,
		ID:             m.ID,
		BaseCurrency:   m.BaseCurrency,
		QuoteCurrency:  m.QuoteCurrency,
		ExchangeID:     m.ExchangeID,
		SymbolCode:     m.SymbolCode,
		Status:         m.Status,
		MinOrderSize:   minSize,
		PricePrecision: pricePrec,
	}
}

func FromSymbolDomain(d *domain.Symbol) *SymbolModel {
	return &SymbolModel{
		Model:          d.Model,
		ID:             d.ID,
		BaseCurrency:   d.BaseCurrency,
		QuoteCurrency:  d.QuoteCurrency,
		ExchangeID:     d.ExchangeID,
		SymbolCode:     d.SymbolCode,
		Status:         d.Status,
		MinOrderSize:   d.MinOrderSize.String(),
		PricePrecision: d.PricePrecision.String(),
	}
}

// ExchangeModel 交易所数据库模型
type ExchangeModel struct {
	gorm.Model
	ID       string `gorm:"column:id;type:varchar(32);primaryKey"`
	Name     string `gorm:"column:name;type:varchar(50);uniqueIndex;not null"`
	Country  string `gorm:"column:country;type:varchar(50)"`
	Status   string `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
	Timezone string `gorm:"column:timezone;type:varchar(50)"`
}

func (ExchangeModel) TableName() string {
	return "exchanges"
}

func (m *ExchangeModel) ToDomain() *domain.Exchange {
	return &domain.Exchange{
		Model:    m.Model,
		ID:       m.ID,
		Name:     m.Name,
		Country:  m.Country,
		Status:   m.Status,
		Timezone: m.Timezone,
	}
}

func FromExchangeDomain(d *domain.Exchange) *ExchangeModel {
	return &ExchangeModel{
		Model:    d.Model,
		ID:       d.ID,
		Name:     d.Name,
		Country:  d.Country,
		Status:   d.Status,
		Timezone: d.Timezone,
	}
}

type symbolRepository struct {
	db *gorm.DB
}

func NewSymbolRepository(db *gorm.DB) domain.SymbolRepository {
	return &symbolRepository{db: db}
}

func (r *symbolRepository) Save(ctx context.Context, symbol *domain.Symbol) error {
	model := FromSymbolDomain(symbol)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *symbolRepository) GetByID(ctx context.Context, id string) (*domain.Symbol, error) {
	var model SymbolModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *symbolRepository) GetByCode(ctx context.Context, code string) (*domain.Symbol, error) {
	var model SymbolModel
	if err := r.db.WithContext(ctx).First(&model, "symbol_code = ?", code).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *symbolRepository) List(ctx context.Context, exchangeID string, status string, limit, offset int) ([]*domain.Symbol, error) {
	var models []SymbolModel
	db := r.db.WithContext(ctx)
	if exchangeID != "" {
		db = db.Where("exchange_id = ?", exchangeID)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if err := db.Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Symbol, len(models))
	for i, m := range models {
		result[i] = m.ToDomain()
	}
	return result, nil
}

type exchangeRepository struct {
	db *gorm.DB
}

func NewExchangeRepository(db *gorm.DB) domain.ExchangeRepository {
	return &exchangeRepository{db: db}
}

func (r *exchangeRepository) Save(ctx context.Context, exchange *domain.Exchange) error {
	model := FromExchangeDomain(exchange)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *exchangeRepository) GetByID(ctx context.Context, id string) (*domain.Exchange, error) {
	var model ExchangeModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *exchangeRepository) List(ctx context.Context, limit, offset int) ([]*domain.Exchange, error) {
	var models []ExchangeModel
	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Exchange, len(models))
	for i, m := range models {
		result[i] = m.ToDomain()
	}
	return result, nil
}
