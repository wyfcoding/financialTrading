package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataManager 处理所有参考数据相关的写入操作（Commands）。
type ReferenceDataManager struct {
	symbolRepo   domain.SymbolRepository
	exchangeRepo domain.ExchangeRepository
}

// NewReferenceDataManager 构造函数。
func NewReferenceDataManager(symbolRepo domain.SymbolRepository, exchangeRepo domain.ExchangeRepository) *ReferenceDataManager {
	return &ReferenceDataManager{
		symbolRepo:   symbolRepo,
		exchangeRepo: exchangeRepo,
	}
}

// SaveSymbol 保存交易对
func (m *ReferenceDataManager) SaveSymbol(ctx context.Context, symbol *domain.Symbol) error {
	return m.symbolRepo.Save(ctx, symbol)
}

// SaveExchange 保存交易所
func (m *ReferenceDataManager) SaveExchange(ctx context.Context, exchange *domain.Exchange) error {
	return m.exchangeRepo.Save(ctx, exchange)
}
