package application

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// CreateSymbolCommand 创建交易对命令
type CreateSymbolCommand struct {
	SymbolID       string
	BaseCurrency   string
	QuoteCurrency  string
	ExchangeID     string
	SymbolCode     string
	Status         string
	MinOrderSize   float64
	PricePrecision float64
}

// UpdateSymbolCommand 更新交易对命令
type UpdateSymbolCommand struct {
	SymbolID       string
	Status         string
	MinOrderSize   float64
	PricePrecision float64
}

// DeleteSymbolCommand 删除交易对命令
type DeleteSymbolCommand struct {
	SymbolID string
}

// CreateExchangeCommand 创建交易所命令
type CreateExchangeCommand struct {
	ExchangeID string
	Name       string
	Country    string
	Status     string
	Timezone   string
}

// UpdateExchangeCommand 更新交易所命令
type UpdateExchangeCommand struct {
	ExchangeID string
	Status     string
	Country    string
	Timezone   string
}

// DeleteExchangeCommand 删除交易所命令
type DeleteExchangeCommand struct {
	ExchangeID string
}

// InstrumentDTO 合约 DTO
type InstrumentDTO struct {
	ID            string `json:"id"`
	Symbol        string `json:"symbol"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	TickSize      string `json:"tick_size"`
	LotSize       string `json:"lot_size"`
	Type          string `json:"type"`
	MaxLeverage   int    `json:"max_leverage"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// SymbolDTO 交易对 DTO
type SymbolDTO struct {
	ID             string `json:"id"`
	BaseCurrency   string `json:"base_currency"`
	QuoteCurrency  string `json:"quote_currency"`
	ExchangeID     string `json:"exchange_id"`
	SymbolCode     string `json:"symbol_code"`
	Status         string `json:"status"`
	MinOrderSize   string `json:"min_order_size"`
	PricePrecision string `json:"price_precision"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

// ExchangeDTO 交易所 DTO
type ExchangeDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Country   string `json:"country"`
	Status    string `json:"status"`
	Timezone  string `json:"timezone"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func toSymbolDTO(s *domain.Symbol) *SymbolDTO {
	if s == nil {
		return nil
	}
	return &SymbolDTO{
		ID:             s.ID,
		BaseCurrency:   s.BaseCurrency,
		QuoteCurrency:  s.QuoteCurrency,
		ExchangeID:     s.ExchangeID,
		SymbolCode:     s.SymbolCode,
		Status:         s.Status,
		MinOrderSize:   s.MinOrderSize.String(),
		PricePrecision: s.PricePrecision.String(),
		CreatedAt:      s.CreatedAt.Unix(),
		UpdatedAt:      s.UpdatedAt.Unix(),
	}
}

func toSymbolDTOs(symbols []*domain.Symbol) []*SymbolDTO {
	dtos := make([]*SymbolDTO, 0, len(symbols))
	for _, s := range symbols {
		dtos = append(dtos, toSymbolDTO(s))
	}
	return dtos
}

func toExchangeDTO(e *domain.Exchange) *ExchangeDTO {
	if e == nil {
		return nil
	}
	return &ExchangeDTO{
		ID:        e.ID,
		Name:      e.Name,
		Country:   e.Country,
		Status:    e.Status,
		Timezone:  e.Timezone,
		CreatedAt: e.CreatedAt.Unix(),
		UpdatedAt: e.UpdatedAt.Unix(),
	}
}

func toExchangeDTOs(exchanges []*domain.Exchange) []*ExchangeDTO {
	dtos := make([]*ExchangeDTO, 0, len(exchanges))
	for _, e := range exchanges {
		dtos = append(dtos, toExchangeDTO(e))
	}
	return dtos
}

func toInstrumentDTO(i *domain.Instrument) *InstrumentDTO {
	if i == nil {
		return nil
	}
	return &InstrumentDTO{
		ID:            i.ID,
		Symbol:        i.Symbol,
		BaseCurrency:  i.BaseCurrency,
		QuoteCurrency: i.QuoteCurrency,
		TickSize:      decimal.NewFromFloat(i.TickSize).String(),
		LotSize:       decimal.NewFromFloat(i.LotSize).String(),
		Type:          string(i.Type),
		MaxLeverage:   i.MaxLeverage,
		CreatedAt:     i.CreatedAt.Unix(),
		UpdatedAt:     i.UpdatedAt.Unix(),
	}
}

func toInstrumentDTOs(instruments []*domain.Instrument) []*InstrumentDTO {
	dtos := make([]*InstrumentDTO, 0, len(instruments))
	for _, i := range instruments {
		dtos = append(dtos, toInstrumentDTO(i))
	}
	return dtos
}
