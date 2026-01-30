package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// ReferenceDataCommandService 处理所有参考数据相关的写入操作（Commands）。
type ReferenceDataCommandService struct {
	repo domain.ReferenceDataRepository
}

// NewReferenceDataCommandService 构造函数。
func NewReferenceDataCommandService(repo domain.ReferenceDataRepository) *ReferenceDataCommandService {
	return &ReferenceDataCommandService{repo: repo}
}

// CreateSymbol 创建交易对
func (s *ReferenceDataCommandService) CreateSymbol(ctx context.Context, base, quote, exchangeID string, minOrderSize, pricePrecision decimal.Decimal) (*domain.Symbol, error) {
	// Simple validation
	if base == "" || quote == "" || exchangeID == "" {
		return nil, fmt.Errorf("invalid arguments")
	}

	symbolCode := fmt.Sprintf("%s%s", base, quote)

	// Check duplications
	existing, _ := s.repo.GetSymbolByCode(ctx, symbolCode)
	if existing != nil {
		return nil, fmt.Errorf("symbol code %s already exists", symbolCode)
	}

	symbol := &domain.Symbol{
		ID:             fmt.Sprintf("SYM-%d", idgen.GenID()),
		BaseCurrency:   base,
		QuoteCurrency:  quote,
		ExchangeID:     exchangeID,
		SymbolCode:     symbolCode,
		Status:         "ACTIVE",
		MinOrderSize:   minOrderSize,
		PricePrecision: pricePrecision,
	}
	symbol.CreatedAt = time.Now()
	symbol.UpdatedAt = time.Now()

	if err := s.repo.SaveSymbol(ctx, symbol); err != nil {
		return nil, err
	}
	return symbol, nil
}

// UpdateSymbolStatus 更新交易对状态
func (s *ReferenceDataCommandService) UpdateSymbolStatus(ctx context.Context, id, status string) error {
	symbol, err := s.repo.GetSymbol(ctx, id)
	if err != nil {
		return err
	}
	if symbol == nil {
		return fmt.Errorf("symbol not found")
	}

	symbol.Status = status
	return s.repo.SaveSymbol(ctx, symbol)
}

// CreateExchange 创建交易所
func (s *ReferenceDataCommandService) CreateExchange(ctx context.Context, name, country, timezone string) (*domain.Exchange, error) {
	exchange := &domain.Exchange{
		ID:       fmt.Sprintf("EX-%d", idgen.GenID()),
		Name:     name,
		Country:  country,
		Timezone: timezone,
		Status:   "ACTIVE",
	}
	exchange.CreatedAt = time.Now()
	exchange.UpdatedAt = time.Now()

	if err := s.repo.SaveExchange(ctx, exchange); err != nil {
		return nil, err
	}
	return exchange, nil
}
