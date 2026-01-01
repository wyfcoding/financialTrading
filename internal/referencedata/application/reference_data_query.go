package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataQuery 处理所有参考数据相关的查询操作（Queries）。
type ReferenceDataQuery struct {
	symbolRepo   domain.SymbolRepository
	exchangeRepo domain.ExchangeRepository
}

// NewReferenceDataQuery 构造函数。
func NewReferenceDataQuery(symbolRepo domain.SymbolRepository, exchangeRepo domain.ExchangeRepository) *ReferenceDataQuery {
	return &ReferenceDataQuery{
		symbolRepo:   symbolRepo,
		exchangeRepo: exchangeRepo,
	}
}

// GetSymbol 获取交易对
func (q *ReferenceDataQuery) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	return q.symbolRepo.GetByID(ctx, id)
}

// ListSymbols 列出交易对
func (q *ReferenceDataQuery) ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	return q.symbolRepo.List(ctx, exchangeID, status, limit, offset)
}

// GetExchange 获取交易所
func (q *ReferenceDataQuery) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	return q.exchangeRepo.GetByID(ctx, id)
}

// ListExchanges 列出交易所
func (q *ReferenceDataQuery) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	return q.exchangeRepo.List(ctx, limit, offset)
}
