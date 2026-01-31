package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataQueryService 处理所有参考数据相关的查询操作（Queries）。
type ReferenceDataQueryService struct {
	repo domain.ReferenceDataRepository
}

// NewReferenceDataQueryService 构造函数。
func NewReferenceDataQueryService(repo domain.ReferenceDataRepository) *ReferenceDataQueryService {
	return &ReferenceDataQueryService{repo: repo}
}

// GetSymbol 获取单个交易对
func (s *ReferenceDataQueryService) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	return s.repo.GetSymbol(ctx, id)
}

// ListSymbols 列表查询
func (s *ReferenceDataQueryService) ListSymbols(ctx context.Context, exchangeID, status string, limit int, offset int) ([]*domain.Symbol, error) {
	return s.repo.ListSymbols(ctx, exchangeID, status, limit, offset)
}

// GetExchange 获取交易所信息
func (s *ReferenceDataQueryService) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	return s.repo.GetExchange(ctx, id)
}

// ListExchanges 交易所列表
func (s *ReferenceDataQueryService) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	return s.repo.ListExchanges(ctx, limit, offset)
}
