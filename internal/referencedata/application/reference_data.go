package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataService 参考数据门面服务，整合 Manager 和 Query。
type ReferenceDataService struct {
	manager *ReferenceDataManager
	query   *ReferenceDataQuery
}

// NewReferenceDataService 构造函数。
func NewReferenceDataService(symbolRepo domain.SymbolRepository, exchangeRepo domain.ExchangeRepository) *ReferenceDataService {
	return &ReferenceDataService{
		manager: NewReferenceDataManager(symbolRepo, exchangeRepo),
		query:   NewReferenceDataQuery(symbolRepo, exchangeRepo),
	}
}

// --- Manager (Writes) ---

func (s *ReferenceDataService) SaveSymbol(ctx context.Context, symbol *domain.Symbol) error {
	return s.manager.SaveSymbol(ctx, symbol)
}

func (s *ReferenceDataService) SaveExchange(ctx context.Context, exchange *domain.Exchange) error {
	return s.manager.SaveExchange(ctx, exchange)
}

// --- Query (Reads) ---

func (s *ReferenceDataService) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	return s.query.GetSymbol(ctx, id)
}

func (s *ReferenceDataService) ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	return s.query.ListSymbols(ctx, exchangeID, status, limit, offset)
}

func (s *ReferenceDataService) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	return s.query.GetExchange(ctx, id)
}

func (s *ReferenceDataService) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	return s.query.ListExchanges(ctx, limit, offset)
}
