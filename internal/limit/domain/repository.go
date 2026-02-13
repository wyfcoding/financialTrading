package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

type LimitRepository interface {
	Save(ctx context.Context, limit *Limit) error
	Update(ctx context.Context, limit *Limit) error
	GetByID(ctx context.Context, id uint64) (*Limit, error)
	GetByLimitID(ctx context.Context, limitID string) (*Limit, error)
	GetByAccountAndType(ctx context.Context, accountID uint64, limitType LimitType, scope LimitScope, symbol string) (*Limit, error)
	ListByAccount(ctx context.Context, accountID uint64) ([]*Limit, error)
	ListExceeded(ctx context.Context) ([]*Limit, error)
	ResetDailyLimits(ctx context.Context) error
}

type LimitBreachRepository interface {
	Save(ctx context.Context, breach *LimitBreach) error
	GetByID(ctx context.Context, id uint64) (*LimitBreach, error)
	ListByAccount(ctx context.Context, accountID uint64, resolved bool, page, pageSize int) ([]*LimitBreach, int64, error)
	ListUnresolved(ctx context.Context, limit int) ([]*LimitBreach, error)
}

type AccountLimitConfigRepository interface {
	Save(ctx context.Context, config *AccountLimitConfig) error
	Update(ctx context.Context, config *AccountLimitConfig) error
	GetByAccountID(ctx context.Context, accountID uint64) (*AccountLimitConfig, error)
}

type LimitService struct {
	limitRepo  LimitRepository
	breachRepo LimitBreachRepository
	configRepo AccountLimitConfigRepository
}

func NewLimitService(limitRepo LimitRepository, breachRepo LimitBreachRepository, configRepo AccountLimitConfigRepository) *LimitService {
	return &LimitService{
		limitRepo:  limitRepo,
		breachRepo: breachRepo,
		configRepo: configRepo,
	}
}

func (s *LimitService) CheckLimit(ctx context.Context, accountID uint64, limitType LimitType, scope LimitScope, symbol string, additionalValue decimal.Decimal) (*LimitCheckResult, error) {
	limit, err := s.limitRepo.GetByAccountAndType(ctx, accountID, limitType, scope, symbol)
	if err != nil {
		return nil, err
	}
	if limit == nil {
		return &LimitCheckResult{Passed: true}, nil
	}

	return limit.CheckLimit(additionalValue)
}

func (s *LimitService) UpdateLimitValue(ctx context.Context, limitID string, value decimal.Decimal) error {
	limit, err := s.limitRepo.GetByLimitID(ctx, limitID)
	if err != nil {
		return err
	}
	if limit == nil {
		return ErrLimitNotFound
	}

	oldStatus := limit.Status
	if err := limit.UpdateCurrentValue(value); err != nil {
		return err
	}

	if limit.Status == LimitStatusExceeded && oldStatus != LimitStatusExceeded {
		breach := NewLimitBreach(
			"BR"+limitID[2:],
			limitID,
			limit.AccountID,
			limit.Type,
			limit.CurrentValue,
			limit.LimitValue,
		)
		s.breachRepo.Save(ctx, breach)
	}

	return s.limitRepo.Update(ctx, limit)
}

func (s *LimitService) InitializeLimitsForAccount(ctx context.Context, accountID uint64) error {
	config, err := s.configRepo.GetByAccountID(ctx, accountID)
	if err != nil {
		config = NewAccountLimitConfig(accountID)
		if err := s.configRepo.Save(ctx, config); err != nil {
			return err
		}
	}

	limits := []struct {
		limitType LimitType
		scope     LimitScope
		value     decimal.Decimal
	}{
		{LimitTypePosition, ScopeAccount, config.MaxPositionSize},
		{LimitTypeOrder, ScopeAccount, config.MaxOrderSize},
		{LimitTypeDaily, ScopeAccount, config.MaxDailyVolume},
		{LimitTypeLoss, ScopeAccount, config.MaxDailyLoss},
		{LimitTypeExposure, ScopeAccount, config.MaxExposure},
		{LimitTypeLeverage, ScopeAccount, config.MaxLeverage},
		{LimitTypeConcentration, ScopeAccount, config.MaxConcentration},
	}

	for _, l := range limits {
		limitID := "LM" + string(l.limitType)[:2] + string(accountID)
		limit := NewLimit(limitID, accountID, l.limitType, l.scope, l.value, config.WarningThreshold)
		if err := s.limitRepo.Save(ctx, limit); err != nil {
			return err
		}
	}

	return nil
}
