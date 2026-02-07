package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type referenceDataRepository struct {
	db *gorm.DB
}

// NewReferenceDataRepository 创建参考数据仓储实例
func NewReferenceDataRepository(db *gorm.DB) domain.ReferenceDataRepository {
	return &referenceDataRepository{db: db}
}

// --- tx helpers ---

func (r *referenceDataRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *referenceDataRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *referenceDataRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *referenceDataRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

// --- Symbol ---

func (r *referenceDataRepository) SaveSymbol(ctx context.Context, symbol *domain.Symbol) error {
	if symbol == nil {
		return nil
	}
	model := toSymbolModel(symbol)
	db := r.getDB(ctx).WithContext(ctx)

	var existing SymbolModel
	query := db
	if symbol.ID != "" {
		query = query.Where("id = ?", symbol.ID)
	} else {
		query = query.Where("symbol_code = ?", symbol.SymbolCode)
	}

	err := query.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *referenceDataRepository) DeleteSymbol(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return r.getDB(ctx).WithContext(ctx).Where("id = ?", id).Delete(&SymbolModel{}).Error
}

func (r *referenceDataRepository) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	var model SymbolModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toSymbol(&model), err
}

func (r *referenceDataRepository) GetSymbolByCode(ctx context.Context, code string) (*domain.Symbol, error) {
	var model SymbolModel
	err := r.getDB(ctx).WithContext(ctx).Where("symbol_code = ?", code).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toSymbol(&model), err
}

func (r *referenceDataRepository) ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	var models []*SymbolModel
	query := r.getDB(ctx).WithContext(ctx)
	if exchangeID != "" {
		query = query.Where("exchange_id = ?", exchangeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Symbol, 0, len(models))
	for _, m := range models {
		result = append(result, toSymbol(m))
	}
	return result, nil
}

// --- Exchange ---

func (r *referenceDataRepository) SaveExchange(ctx context.Context, exchange *domain.Exchange) error {
	if exchange == nil {
		return nil
	}
	model := toExchangeModel(exchange)
	db := r.getDB(ctx).WithContext(ctx)

	var existing ExchangeModel
	query := db
	if exchange.ID != "" {
		query = query.Where("id = ?", exchange.ID)
	} else {
		query = query.Where("name = ?", exchange.Name)
	}

	err := query.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *referenceDataRepository) DeleteExchange(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return r.getDB(ctx).WithContext(ctx).Where("id = ?", id).Delete(&ExchangeModel{}).Error
}

func (r *referenceDataRepository) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	var model ExchangeModel
	err := r.getDB(ctx).WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toExchange(&model), err
}

func (r *referenceDataRepository) GetExchangeByName(ctx context.Context, name string) (*domain.Exchange, error) {
	var model ExchangeModel
	err := r.getDB(ctx).WithContext(ctx).Where("name = ?", name).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toExchange(&model), err
}

func (r *referenceDataRepository) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	var models []*ExchangeModel
	err := r.getDB(ctx).WithContext(ctx).Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Exchange, 0, len(models))
	for _, m := range models {
		result = append(result, toExchange(m))
	}
	return result, nil
}

// --- Instrument ---

func (r *referenceDataRepository) SaveInstrument(ctx context.Context, instrument *domain.Instrument) error {
	if instrument == nil {
		return nil
	}
	model := toInstrumentModel(instrument)
	db := r.getDB(ctx).WithContext(ctx)

	var existing InstrumentModel
	query := db
	if instrument.ID != "" {
		query = query.Where("id = ?", instrument.ID)
	} else {
		query = query.Where("symbol = ?", instrument.Symbol)
	}

	err := query.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(model).Error
	}
	if err != nil {
		return err
	}

	model.ID = existing.ID
	model.CreatedAt = existing.CreatedAt
	return db.Save(model).Error
}

func (r *referenceDataRepository) DeleteInstrument(ctx context.Context, symbol string) error {
	if symbol == "" {
		return nil
	}
	return r.getDB(ctx).WithContext(ctx).Where("symbol = ?", symbol).Delete(&InstrumentModel{}).Error
}

func (r *referenceDataRepository) GetInstrument(ctx context.Context, symbol string) (*domain.Instrument, error) {
	var model InstrumentModel
	err := r.getDB(ctx).WithContext(ctx).Where("symbol = ?", symbol).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return toInstrument(&model), err
}

func (r *referenceDataRepository) ListInstruments(ctx context.Context, limit int, offset int) ([]*domain.Instrument, error) {
	var models []*InstrumentModel
	err := r.getDB(ctx).WithContext(ctx).Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Instrument, 0, len(models))
	for _, m := range models {
		result = append(result, toInstrument(m))
	}
	return result, nil
}

func (r *referenceDataRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
