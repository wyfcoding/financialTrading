package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"gorm.io/gorm"
)

type referenceDataRepository struct {
	db *gorm.DB
}

// NewReferenceDataRepository 创建参考数据仓储实例
func NewReferenceDataRepository(db *gorm.DB) domain.ReferenceDataRepository {
	return &referenceDataRepository{db: db}
}

// --- Symbol ---

func (r *referenceDataRepository) SaveSymbol(ctx context.Context, symbol *domain.Symbol) error {
	var existing domain.Symbol
	err := r.db.WithContext(ctx).Where("symbol_code = ?", symbol.SymbolCode).First(&existing).Error
	if err == nil {
		symbol.ID = existing.ID
		symbol.CreatedAt = existing.CreatedAt
		return r.db.WithContext(ctx).Save(symbol).Error
	}
	return r.db.WithContext(ctx).Create(symbol).Error
}

func (r *referenceDataRepository) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	var symbol domain.Symbol
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&symbol).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &symbol, err
}

func (r *referenceDataRepository) GetSymbolByCode(ctx context.Context, code string) (*domain.Symbol, error) {
	var symbol domain.Symbol
	err := r.db.WithContext(ctx).Where("symbol_code = ?", code).First(&symbol).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &symbol, err
}

func (r *referenceDataRepository) ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	var symbols []*domain.Symbol
	query := r.db.WithContext(ctx)
	if exchangeID != "" {
		query = query.Where("exchange_id = ?", exchangeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Limit(limit).Offset(offset).Find(&symbols).Error
	return symbols, err
}

// --- Exchange ---

func (r *referenceDataRepository) SaveExchange(ctx context.Context, exchange *domain.Exchange) error {
	var existing domain.Exchange
	err := r.db.WithContext(ctx).Where("name = ?", exchange.Name).First(&existing).Error
	if err == nil {
		exchange.ID = existing.ID
		exchange.CreatedAt = existing.CreatedAt
		return r.db.WithContext(ctx).Save(exchange).Error
	}
	return r.db.WithContext(ctx).Create(exchange).Error
}

func (r *referenceDataRepository) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	var exchange domain.Exchange
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&exchange).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &exchange, err
}

func (r *referenceDataRepository) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	var exchanges []*domain.Exchange
	err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&exchanges).Error
	return exchanges, err
}
