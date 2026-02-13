package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/limit/domain"
)

type CheckLimitCommand struct {
	AccountID        uint64
	LimitType        domain.LimitType
	Scope            domain.LimitScope
	Symbol           string
	AdditionalValue  decimal.Decimal
}

type CheckLimitResult struct {
	Passed       bool            `json:"passed"`
	Warning      bool            `json:"warning"`
	CurrentValue decimal.Decimal `json:"current_value"`
	LimitValue   decimal.Decimal `json:"limit_value"`
	UsedPercent  decimal.Decimal `json:"used_percent"`
	Reason       string          `json:"reason"`
}

type UpdateLimitValueCommand struct {
	LimitID string
	Value   decimal.Decimal
}

type InitializeAccountLimitsCommand struct {
	AccountID uint64
}

type LimitApplicationService struct {
	limitRepo  domain.LimitRepository
	breachRepo domain.LimitBreachRepository
	configRepo domain.AccountLimitConfigRepository
	limitSvc   *domain.LimitService
	logger     *slog.Logger
}

func NewLimitApplicationService(
	limitRepo domain.LimitRepository,
	breachRepo domain.LimitBreachRepository,
	configRepo domain.AccountLimitConfigRepository,
	logger *slog.Logger,
) *LimitApplicationService {
	return &LimitApplicationService{
		limitRepo:  limitRepo,
		breachRepo: breachRepo,
		configRepo: configRepo,
		limitSvc:   domain.NewLimitService(limitRepo, breachRepo, configRepo),
		logger:     logger,
	}
}

func (s *LimitApplicationService) CheckLimit(ctx context.Context, cmd *CheckLimitCommand) (*CheckLimitResult, error) {
	result, err := s.limitSvc.CheckLimit(ctx, cmd.AccountID, cmd.LimitType, cmd.Scope, cmd.Symbol, cmd.AdditionalValue)
	if err != nil {
		return nil, err
	}

	return &CheckLimitResult{
		Passed:       result.Passed,
		Warning:      result.Warning,
		CurrentValue: result.CurrentValue,
		LimitValue:   result.LimitValue,
		UsedPercent:  result.UsedPercent,
		Reason:       result.Reason,
	}, nil
}

func (s *LimitApplicationService) UpdateLimitValue(ctx context.Context, cmd *UpdateLimitValueCommand) error {
	return s.limitSvc.UpdateLimitValue(ctx, cmd.LimitID, cmd.Value)
}

func (s *LimitApplicationService) InitializeAccountLimits(ctx context.Context, cmd *InitializeAccountLimitsCommand) error {
	return s.limitSvc.InitializeLimitsForAccount(ctx, cmd.AccountID)
}

func (s *LimitApplicationService) GetLimit(ctx context.Context, limitID string) (*domain.Limit, error) {
	return s.limitRepo.GetByLimitID(ctx, limitID)
}

func (s *LimitApplicationService) ListAccountLimits(ctx context.Context, accountID uint64) ([]*domain.Limit, error) {
	return s.limitRepo.ListByAccount(ctx, accountID)
}

func (s *LimitApplicationService) GetLimitConfig(ctx context.Context, accountID uint64) (*domain.AccountLimitConfig, error) {
	return s.configRepo.GetByAccountID(ctx, accountID)
}

func (s *LimitApplicationService) UpdateLimitConfig(ctx context.Context, config *domain.AccountLimitConfig) error {
	existing, err := s.configRepo.GetByAccountID(ctx, config.AccountID)
	if err != nil {
		return err
	}
	if existing == nil {
		return s.configRepo.Save(ctx, config)
	}
	return s.configRepo.Update(ctx, config)
}

func (s *LimitApplicationService) ListBreaches(ctx context.Context, accountID uint64, resolved bool, page, pageSize int) ([]*domain.LimitBreach, int64, error) {
	return s.breachRepo.ListByAccount(ctx, accountID, resolved, page, pageSize)
}

func (s *LimitApplicationService) ResolveBreach(ctx context.Context, breachID uint64) error {
	breach, err := s.breachRepo.GetByID(ctx, breachID)
	if err != nil {
		return err
	}
	if breach == nil {
		return domain.ErrBreachNotFound
	}

	breach.Resolve()
	return s.breachRepo.Save(ctx, breach)
}

func (s *LimitApplicationService) FreezeLimit(ctx context.Context, limitID string) error {
	limit, err := s.limitRepo.GetByLimitID(ctx, limitID)
	if err != nil {
		return err
	}
	if limit == nil {
		return domain.ErrLimitNotFound
	}

	limit.Freeze()
	return s.limitRepo.Update(ctx, limit)
}

func (s *LimitApplicationService) UnfreezeLimit(ctx context.Context, limitID string) error {
	limit, err := s.limitRepo.GetByLimitID(ctx, limitID)
	if err != nil {
		return err
	}
	if limit == nil {
		return domain.ErrLimitNotFound
	}

	limit.Unfreeze()
	return s.limitRepo.Update(ctx, limit)
}

func (s *LimitApplicationService) ResetDailyLimits(ctx context.Context) error {
	return s.limitRepo.ResetDailyLimits(ctx)
}

func (s *LimitApplicationService) CreateInstrumentLimit(ctx context.Context, accountID uint64, symbol string, limitType domain.LimitType, limitValue decimal.Decimal) error {
	config, err := s.configRepo.GetByAccountID(ctx, accountID)
	if err != nil || config == nil {
		config = domain.NewAccountLimitConfig(accountID)
		s.configRepo.Save(ctx, config)
	}

	limitID := fmt.Sprintf("LM%s%s%d", limitType[:2], symbol[:min(4, len(symbol))], accountID)
	limit := domain.NewLimit(limitID, accountID, limitType, domain.ScopeInstrument, limitValue, config.WarningThreshold)
	limit.SetSymbol(symbol)

	return s.limitRepo.Save(ctx, limit)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type LimitDTO struct {
	ID               uint64  `json:"id"`
	LimitID          string  `json:"limit_id"`
	AccountID        uint64  `json:"account_id"`
	Type             string  `json:"type"`
	Scope            string  `json:"scope"`
	Symbol           string  `json:"symbol"`
	CurrentValue     string  `json:"current_value"`
	LimitValue       string  `json:"limit_value"`
	WarningThreshold string  `json:"warning_threshold"`
	UsedPercent      string  `json:"used_percent"`
	Status           string  `json:"status"`
	IsActive         bool    `json:"is_active"`
}

func ToLimitDTO(l *domain.Limit) *LimitDTO {
	return &LimitDTO{
		ID:               l.ID,
		LimitID:          l.LimitID,
		AccountID:        l.AccountID,
		Type:             string(l.Type),
		Scope:            string(l.Scope),
		Symbol:           l.Symbol,
		CurrentValue:     l.CurrentValue.String(),
		LimitValue:       l.LimitValue.String(),
		WarningThreshold: l.WarningThreshold.String(),
		UsedPercent:      l.UsedPercent.String(),
		Status:           string(l.Status),
		IsActive:         l.IsActive,
	}
}

type BreachDTO struct {
	ID            uint64 `json:"id"`
	BreachID      string `json:"breach_id"`
	LimitID       string `json:"limit_id"`
	AccountID     uint64 `json:"account_id"`
	Type          string `json:"type"`
	CurrentValue  string `json:"current_value"`
	LimitValue    string `json:"limit_value"`
	BreachPercent string `json:"breach_percent"`
	Action        string `json:"action"`
	Resolved      bool   `json:"resolved"`
	CreatedAt     string `json:"created_at"`
}

func ToBreachDTO(b *domain.LimitBreach) *BreachDTO {
	return &BreachDTO{
		ID:            b.ID,
		BreachID:      b.BreachID,
		LimitID:       b.LimitID,
		AccountID:     b.AccountID,
		Type:          string(b.Type),
		CurrentValue:  b.CurrentValue.String(),
		LimitValue:    b.LimitValue.String(),
		BreachPercent: b.BreachPercent.String(),
		Action:        b.Action,
		Resolved:      b.ResolvedAt != nil,
		CreatedAt:     b.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
