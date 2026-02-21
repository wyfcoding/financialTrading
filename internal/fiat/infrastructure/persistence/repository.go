package persistence

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/fiat/domain"
	"gorm.io/gorm"
)

// ExchangeRateModel GORM 模型
type ExchangeRateModel struct {
	ID           uint64          `gorm:"primaryKey;autoIncrement"`
	FromCurrency string          `gorm:"type:varchar(10);index:idx_pair,unique"`
	ToCurrency   string          `gorm:"type:varchar(10);index:idx_pair,unique"`
	BaseRate     decimal.Decimal `gorm:"type:decimal(20,8);not null"`
	BidRate      decimal.Decimal `gorm:"type:decimal(20,8);not null"`
	AskRate      decimal.Decimal `gorm:"type:decimal(20,8);not null"`
	Source       string          `gorm:"type:varchar(50)"`
	UpdatedAt    time.Time
}

func (ExchangeRateModel) TableName() string {
	return "fiat_exchange_rates"
}

// RateLockModel GORM 模型
type RateLockModel struct {
	gorm.Model
	LockID       string          `gorm:"type:varchar(100);uniqueIndex"`
	UserID       string          `gorm:"type:varchar(100);index"`
	PaymentID    string          `gorm:"type:varchar(100);index"`
	FromCurrency string          `gorm:"type:varchar(10)"`
	ToCurrency   string          `gorm:"type:varchar(10)"`
	LockedRate   decimal.Decimal `gorm:"type:decimal(20,8)"`
	Amount       decimal.Decimal `gorm:"type:decimal(20,8)"`
	LockedAmount decimal.Decimal `gorm:"type:decimal(20,8)"`
	Status       string          `gorm:"type:varchar(20)"`
	ExpiresAt    time.Time
}

func (RateLockModel) TableName() string {
	return "fiat_rate_locks"
}

type fiatRepository struct {
	db *gorm.DB
}

func NewFiatRepository(db *gorm.DB) domain.FiatRepository {
	return &fiatRepository{db: db}
}

func (r *fiatRepository) SaveRate(ctx context.Context, rate *domain.ExchangeRate) error {
	m := &ExchangeRateModel{
		FromCurrency: rate.FromCurrency,
		ToCurrency:   rate.ToCurrency,
		BaseRate:     rate.BaseRate,
		BidRate:      rate.BidRate,
		AskRate:      rate.AskRate,
		Source:       rate.Source,
		UpdatedAt:    rate.UpdatedAt,
	}
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *fiatRepository) GetRate(ctx context.Context, from, to string) (*domain.ExchangeRate, error) {
	var m ExchangeRateModel
	err := r.db.WithContext(ctx).Where("from_currency = ? AND to_currency = ?", from, to).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果没找到，尝试在代码中插入一些默认汇率进行演示
			return r.getMockRate(from, to)
		}
		return nil, err
	}
	return &domain.ExchangeRate{
		ID:           m.ID,
		FromCurrency: m.FromCurrency,
		ToCurrency:   m.ToCurrency,
		BaseRate:     m.BaseRate,
		BidRate:      m.BidRate,
		AskRate:      m.AskRate,
		Source:       m.Source,
		UpdatedAt:    m.UpdatedAt,
	}, nil
}

func (r *fiatRepository) getMockRate(from, to string) (*domain.ExchangeRate, error) {
	// 简单模拟
	rates := map[string]float64{
		"USD/CNY": 7.23,
		"CNY/USD": 0.138,
		"EUR/CNY": 7.85,
		"CNY/EUR": 0.127,
	}
	pair := from + "/" + to
	val, ok := rates[pair]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &domain.ExchangeRate{
		FromCurrency: from,
		ToCurrency:   to,
		BaseRate:     decimal.NewFromFloat(val),
		BidRate:      decimal.NewFromFloat(val * 0.998),
		AskRate:      decimal.NewFromFloat(val * 1.002),
		Source:       "MOCK",
		UpdatedAt:    time.Now(),
	}, nil
}

func (r *fiatRepository) SaveLock(ctx context.Context, lock *domain.RateLock) error {
	m := &RateLockModel{
		LockID:       lock.ID,
		UserID:       lock.UserID,
		PaymentID:    lock.PaymentID,
		FromCurrency: lock.FromCurrency,
		ToCurrency:   lock.ToCurrency,
		LockedRate:   lock.LockedRate,
		Amount:       lock.Amount,
		LockedAmount: lock.LockedAmount,
		Status:       string(lock.Status),
		ExpiresAt:    lock.ExpiresAt,
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *fiatRepository) GetLock(ctx context.Context, lockID string) (*domain.RateLock, error) {
	var m RateLockModel
	if err := r.db.WithContext(ctx).Where("lock_id = ?", lockID).First(&m).Error; err != nil {
		return nil, err
	}
	return &domain.RateLock{
		ID:           m.LockID,
		UserID:       m.UserID,
		PaymentID:    m.PaymentID,
		FromCurrency: m.FromCurrency,
		ToCurrency:   m.ToCurrency,
		LockedRate:   m.LockedRate,
		Amount:       m.Amount,
		LockedAmount: m.LockedAmount,
		Status:       domain.LockStatus(m.Status),
		ExpiresAt:    m.ExpiresAt,
		CreatedAt:    m.CreatedAt,
	}, nil
}

func (r *fiatRepository) UpdateLock(ctx context.Context, lock *domain.RateLock) error {
	return r.db.WithContext(ctx).Model(&RateLockModel{}).
		Where("lock_id = ?", lock.ID).
		Update("status", string(lock.Status)).Error
}
