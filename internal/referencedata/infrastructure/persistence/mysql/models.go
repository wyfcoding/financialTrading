package mysql

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"gorm.io/gorm"
)

// SymbolModel MySQL 交易对表映射
type SymbolModel struct {
	gorm.Model
	ID             string          `gorm:"primaryKey;type:varchar(32);column:id"`
	BaseCurrency   string          `gorm:"column:base_currency;type:varchar(10);not null"`
	QuoteCurrency  string          `gorm:"column:quote_currency;type:varchar(10);not null"`
	ExchangeID     string          `gorm:"column:exchange_id;type:varchar(32);index;not null"`
	SymbolCode     string          `gorm:"column:symbol_code;type:varchar(20);uniqueIndex;not null"`
	Status         string          `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
	MinOrderSize   decimal.Decimal `gorm:"column:min_order_size;type:decimal(20,8);default:0"`
	PricePrecision decimal.Decimal `gorm:"column:price_precision;type:decimal(20,8);default:0"`
}

func (SymbolModel) TableName() string { return "symbols" }

// ExchangeModel MySQL 交易所表映射
type ExchangeModel struct {
	gorm.Model
	ID       string `gorm:"primaryKey;type:varchar(32);column:id"`
	Name     string `gorm:"column:name;type:varchar(50);uniqueIndex;not null"`
	Country  string `gorm:"column:country;type:varchar(50)"`
	Status   string `gorm:"column:status;type:varchar(20);default:'ACTIVE'"`
	Timezone string `gorm:"column:timezone;type:varchar(50)"`
}

func (ExchangeModel) TableName() string { return "exchanges" }

// InstrumentModel MySQL 合约表映射
type InstrumentModel struct {
	gorm.Model
	ID            string  `gorm:"primaryKey;type:varchar(32);column:id"`
	Symbol        string  `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null"`
	BaseCurrency  string  `gorm:"column:base_currency;type:varchar(10);not null"`
	QuoteCurrency string  `gorm:"column:quote_currency;type:varchar(10);not null"`
	TickSize      float64 `gorm:"column:tick_size;type:decimal(20,8);not null"`
	LotSize       float64 `gorm:"column:lot_size;type:decimal(20,8);not null"`
	Type          string  `gorm:"column:type;type:varchar(10);not null"`
	MaxLeverage   int     `gorm:"column:max_leverage;default:1"`
}

func (InstrumentModel) TableName() string { return "instruments" }

// --- mapping helpers ---

func toSymbolModel(s *domain.Symbol) *SymbolModel {
	if s == nil {
		return nil
	}
	return &SymbolModel{
		ID:             s.ID,
		BaseCurrency:   s.BaseCurrency,
		QuoteCurrency:  s.QuoteCurrency,
		ExchangeID:     s.ExchangeID,
		SymbolCode:     s.SymbolCode,
		Status:         s.Status,
		MinOrderSize:   s.MinOrderSize,
		PricePrecision: s.PricePrecision,
	}
}

func toSymbol(m *SymbolModel) *domain.Symbol {
	if m == nil {
		return nil
	}
	return &domain.Symbol{
		ID:             m.ID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		BaseCurrency:   m.BaseCurrency,
		QuoteCurrency:  m.QuoteCurrency,
		ExchangeID:     m.ExchangeID,
		SymbolCode:     m.SymbolCode,
		Status:         m.Status,
		MinOrderSize:   m.MinOrderSize,
		PricePrecision: m.PricePrecision,
	}
}

func toExchangeModel(e *domain.Exchange) *ExchangeModel {
	if e == nil {
		return nil
	}
	return &ExchangeModel{
		ID:       e.ID,
		Name:     e.Name,
		Country:  e.Country,
		Status:   e.Status,
		Timezone: e.Timezone,
	}
}

func toExchange(m *ExchangeModel) *domain.Exchange {
	if m == nil {
		return nil
	}
	return &domain.Exchange{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Name:      m.Name,
		Country:   m.Country,
		Status:    m.Status,
		Timezone:  m.Timezone,
	}
}

func toInstrumentModel(i *domain.Instrument) *InstrumentModel {
	if i == nil {
		return nil
	}
	return &InstrumentModel{
		ID:            i.ID,
		Symbol:        i.Symbol,
		BaseCurrency:  i.BaseCurrency,
		QuoteCurrency: i.QuoteCurrency,
		TickSize:      i.TickSize,
		LotSize:       i.LotSize,
		Type:          string(i.Type),
		MaxLeverage:   i.MaxLeverage,
	}
}

func toInstrument(m *InstrumentModel) *domain.Instrument {
	if m == nil {
		return nil
	}
	return &domain.Instrument{
		ID:            m.ID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		Symbol:        m.Symbol,
		BaseCurrency:  m.BaseCurrency,
		QuoteCurrency: m.QuoteCurrency,
		TickSize:      m.TickSize,
		LotSize:       m.LotSize,
		Type:          domain.InstrumentType(m.Type),
		MaxLeverage:   m.MaxLeverage,
	}
}
