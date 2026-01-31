package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataService 参考数据门面服务。
type ReferenceDataService struct {
	Command *ReferenceDataCommandService
	Query   *ReferenceDataQueryService
}

// NewReferenceDataService 构造函数。
func NewReferenceDataService(repo domain.ReferenceDataRepository) *ReferenceDataService {
	return &ReferenceDataService{
		Command: NewReferenceDataCommandService(repo),
		Query:   NewReferenceDataQueryService(repo),
	}
}

// --- Command Facade ---

func (s *ReferenceDataService) CreateSymbol(ctx context.Context, base, quote, exchangeID string, minOrderSize, pricePrecision decimal.Decimal) (*domain.Symbol, error) {
	return s.Command.CreateSymbol(ctx, base, quote, exchangeID, minOrderSize, pricePrecision)
}

func (s *ReferenceDataService) UpdateSymbolStatus(ctx context.Context, id, status string) error {
	return s.Command.UpdateSymbolStatus(ctx, id, status)
}

func (s *ReferenceDataService) CreateExchange(ctx context.Context, name, country, timezone string) (*domain.Exchange, error) {
	return s.Command.CreateExchange(ctx, name, country, timezone)
}

// --- Query Facade ---

func (s *ReferenceDataService) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	return s.Query.GetSymbol(ctx, id)
}

func (s *ReferenceDataService) ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	return s.Query.ListSymbols(ctx, exchangeID, status, limit, offset)
}

func (s *ReferenceDataService) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	return s.Query.GetExchange(ctx, id)
}

func (s *ReferenceDataService) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	return s.Query.ListExchanges(ctx, limit, offset)
}
