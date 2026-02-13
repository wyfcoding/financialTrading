package mysql

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
	"gorm.io/gorm"
)

type SettlementRepo struct {
	db *gorm.DB
}

func NewSettlementRepo(db *gorm.DB) domain.SettlementRepository {
	return &SettlementRepo{db: db}
}

func (r *SettlementRepo) Save(ctx context.Context, instruction *domain.SettlementInstruction) error {
	return r.db.WithContext(ctx).Create(instruction).Error
}

func (r *SettlementRepo) Update(ctx context.Context, instruction *domain.SettlementInstruction) error {
	return r.db.WithContext(ctx).Save(instruction).Error
}

func (r *SettlementRepo) Get(ctx context.Context, instructionID string) (*domain.SettlementInstruction, error) {
	var instruction domain.SettlementInstruction
	if err := r.db.WithContext(ctx).
		Where("instruction_id = ?", instructionID).
		Preload("Events").
		First(&instruction).Error; err != nil {
		return nil, err
	}
	return &instruction, nil
}

func (r *SettlementRepo) GetByTradeID(ctx context.Context, tradeID string) (*domain.SettlementInstruction, error) {
	var instruction domain.SettlementInstruction
	if err := r.db.WithContext(ctx).
		Where("trade_id = ?", tradeID).
		Preload("Events").
		First(&instruction).Error; err != nil {
		return nil, err
	}
	return &instruction, nil
}

func (r *SettlementRepo) FindPendingByDate(ctx context.Context, date time.Time, limit int) ([]*domain.SettlementInstruction, error) {
	var instructions []*domain.SettlementInstruction
	err := r.db.WithContext(ctx).
		Where("status = ? AND settlement_date <= ?", domain.SettlementStatusPending, date).
		Order("settlement_date ASC, created_at ASC").
		Limit(limit).
		Find(&instructions).Error
	return instructions, err
}

func (r *SettlementRepo) FindPendingByAccount(ctx context.Context, accountID string, limit int) ([]*domain.SettlementInstruction, error) {
	var instructions []*domain.SettlementInstruction
	err := r.db.WithContext(ctx).
		Where("status = ? AND (buyer_account_id = ? OR seller_account_id = ?)",
			domain.SettlementStatusPending, accountID, accountID).
		Order("settlement_date ASC, created_at ASC").
		Limit(limit).
		Find(&instructions).Error
	return instructions, err
}

func (r *SettlementRepo) UpdateStatus(ctx context.Context, instructionID string, status domain.SettlementStatus, reason string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if reason != "" {
		updates["fail_reason"] = reason
	}
	if status == domain.SettlementStatusSettled {
		updates["settled_at"] = time.Now()
	}

	return r.db.WithContext(ctx).Model(&domain.SettlementInstruction{}).
		Where("instruction_id = ?", instructionID).
		Updates(updates).Error
}

func (r *SettlementRepo) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx)
	})
}

type NettingRepo struct {
	db *gorm.DB
}

func NewNettingRepo(db *gorm.DB) domain.NettingRepository {
	return &NettingRepo{db: db}
}

func (r *NettingRepo) Save(ctx context.Context, result *domain.NettingResult) error {
	return r.db.WithContext(ctx).Save(result).Error
}

func (r *NettingRepo) Get(ctx context.Context, nettingID string) (*domain.NettingResult, error) {
	var result domain.NettingResult
	if err := r.db.WithContext(ctx).Where("netting_id = ?", nettingID).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *NettingRepo) GetByAccountAndCurrency(ctx context.Context, accountID, currency string) (*domain.NettingResult, error) {
	var result domain.NettingResult
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND currency = ? AND status = ?", accountID, currency, "PENDING").
		First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

type BatchRepo struct {
	db *gorm.DB
}

func NewBatchRepo(db *gorm.DB) domain.BatchRepository {
	return &BatchRepo{db: db}
}

func (r *BatchRepo) Save(ctx context.Context, batch *domain.SettlementBatch) error {
	return r.db.WithContext(ctx).Save(batch).Error
}

func (r *BatchRepo) Get(ctx context.Context, batchID string) (*domain.SettlementBatch, error) {
	var batch domain.SettlementBatch
	if err := r.db.WithContext(ctx).Where("batch_id = ?", batchID).First(&batch).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

func (r *BatchRepo) GetByDate(ctx context.Context, date time.Time) (*domain.SettlementBatch, error) {
	var batch domain.SettlementBatch
	if err := r.db.WithContext(ctx).
		Where("settlement_date = ?", date).
		First(&batch).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

type FXRateRepo struct {
	db *gorm.DB
}

func NewFXRateRepo(db *gorm.DB) domain.FXRateRepository {
	return &FXRateRepo{db: db}
}

func (r *FXRateRepo) GetRate(ctx context.Context, fromCurrency, toCurrency string) (*domain.FXRate, error) {
	var rate domain.FXRate
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ? AND effective_at <= ? AND (expires_at IS NULL OR expires_at > ?)",
			fromCurrency, toCurrency, now, now).
		Order("effective_at DESC").
		First(&rate).Error; err != nil {
		return nil, err
	}
	return &rate, nil
}

func (r *FXRateRepo) SaveRate(ctx context.Context, rate *domain.FXRate) error {
	return r.db.WithContext(ctx).Save(rate).Error
}
