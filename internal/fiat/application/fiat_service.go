package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/fiat/domain"
)

type FiatApplicationService struct {
	repo domain.FiatRepository
}

func NewFiatApplicationService(repo domain.FiatRepository) *FiatApplicationService {
	return &FiatApplicationService{repo: repo}
}

// GetRate 获取汇率
func (s *FiatApplicationService) GetRate(ctx context.Context, from, to string) (float64, error) {
	rate, err := s.repo.GetRate(ctx, from, to)
	if err != nil {
		// 这里可以调用外部接口获取实时汇率并缓存
		return 0, err
	}
	f, _ := rate.BaseRate.Float64()
	return f, nil
}

type ExchangeCommand struct {
	FromCurrency string
	ToCurrency   string
	Amount       decimal.Decimal
}

type ExchangeResult struct {
	FromAmount   decimal.Decimal
	ToAmount     decimal.Decimal
	Rate         decimal.Decimal
	FromCurrency string
	ToCurrency   string
}

// Exchange 货币兑换
func (s *FiatApplicationService) Exchange(ctx context.Context, cmd *ExchangeCommand) (*ExchangeResult, error) {
	rate, err := s.repo.GetRate(ctx, cmd.FromCurrency, cmd.ToCurrency)
	if err != nil {
		return nil, err
	}

	exchangedAmount := cmd.Amount.Mul(rate.AskRate) // 使用卖出价

	return &ExchangeResult{
		FromAmount:   cmd.Amount,
		ToAmount:     exchangedAmount,
		Rate:         rate.AskRate,
		FromCurrency: cmd.FromCurrency,
		ToCurrency:   cmd.ToCurrency,
	}, nil
}

type LockRateCommand struct {
	UserID       string
	PaymentID    string
	FromCurrency string
	ToCurrency   string
	Amount       decimal.Decimal
}

// LockRate 锁定汇率
func (s *FiatApplicationService) LockRate(ctx context.Context, cmd *LockRateCommand) (*domain.RateLock, error) {
	rate, err := s.repo.GetRate(ctx, cmd.FromCurrency, cmd.ToCurrency)
	if err != nil {
		return nil, err
	}

	lockID := uuid.New().String()
	lockedAmount := cmd.Amount.Mul(rate.AskRate)

	lock := &domain.RateLock{
		ID:           lockID,
		UserID:       cmd.UserID,
		PaymentID:    cmd.PaymentID,
		FromCurrency: cmd.FromCurrency,
		ToCurrency:   cmd.ToCurrency,
		LockedRate:   rate.AskRate,
		Amount:       cmd.Amount,
		LockedAmount: lockedAmount,
		Status:       domain.LockStatusActive,
		ExpiresAt:    time.Now().Add(15 * time.Minute), // 锁定15分钟
		CreatedAt:    time.Now(),
	}

	if err := s.repo.SaveLock(ctx, lock); err != nil {
		return nil, err
	}

	return lock, nil
}

// VerifyLock 验证锁定
func (s *FiatApplicationService) VerifyLock(ctx context.Context, lockID string) (*domain.RateLock, error) {
	lock, err := s.repo.GetLock(ctx, lockID)
	if err != nil {
		return nil, err
	}

	if !lock.IsValid() {
		return nil, fmt.Errorf("lock expired or invalid")
	}

	return lock, nil
}
