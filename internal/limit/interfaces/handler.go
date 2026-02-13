package interfaces

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/limit/application"
	"github.com/wyfcoding/financialtrading/internal/limit/domain"
)

type LimitHandler struct {
	app *application.LimitApplicationService
}

func NewLimitHandler(app *application.LimitApplicationService) *LimitHandler {
	return &LimitHandler{app: app}
}

type CheckLimitRequest struct {
	AccountID       uint64 `json:"account_id"`
	LimitType       string `json:"limit_type"`
	Scope           string `json:"scope"`
	Symbol          string `json:"symbol"`
	AdditionalValue string `json:"additional_value"`
}

type CheckLimitResponse struct {
	Passed       bool   `json:"passed"`
	Warning      bool   `json:"warning"`
	CurrentValue string `json:"current_value"`
	LimitValue   string `json:"limit_value"`
	UsedPercent  string `json:"used_percent"`
	Reason       string `json:"reason"`
}

func (h *LimitHandler) CheckLimit(ctx context.Context, req *CheckLimitRequest) (*CheckLimitResponse, error) {
	additionalValue, _ := decimal.NewFromString(req.AdditionalValue)

	result, err := h.app.CheckLimit(ctx, &application.CheckLimitCommand{
		AccountID:       req.AccountID,
		LimitType:       domain.LimitType(req.LimitType),
		Scope:           domain.LimitScope(req.Scope),
		Symbol:          req.Symbol,
		AdditionalValue: additionalValue,
	})
	if err != nil {
		return nil, err
	}

	return &CheckLimitResponse{
		Passed:       result.Passed,
		Warning:      result.Warning,
		CurrentValue: result.CurrentValue.String(),
		LimitValue:   result.LimitValue.String(),
		UsedPercent:  result.UsedPercent.String(),
		Reason:       result.Reason,
	}, nil
}

type UpdateLimitValueRequest struct {
	LimitID string `json:"limit_id"`
	Value   string `json:"value"`
}

func (h *LimitHandler) UpdateLimitValue(ctx context.Context, req *UpdateLimitValueRequest) error {
	value, _ := decimal.NewFromString(req.Value)
	return h.app.UpdateLimitValue(ctx, &application.UpdateLimitValueCommand{
		LimitID: req.LimitID,
		Value:   value,
	})
}

type InitializeAccountLimitsRequest struct {
	AccountID uint64 `json:"account_id"`
}

func (h *LimitHandler) InitializeAccountLimits(ctx context.Context, req *InitializeAccountLimitsRequest) error {
	return h.app.InitializeAccountLimits(ctx, &application.InitializeAccountLimitsCommand{
		AccountID: req.AccountID,
	})
}

type GetLimitRequest struct {
	LimitID string `json:"limit_id"`
}

func (h *LimitHandler) GetLimit(ctx context.Context, req *GetLimitRequest) (*application.LimitDTO, error) {
	limit, err := h.app.GetLimit(ctx, req.LimitID)
	if err != nil {
		return nil, err
	}
	if limit == nil {
		return nil, domain.ErrLimitNotFound
	}
	return application.ToLimitDTO(limit), nil
}

type ListAccountLimitsRequest struct {
	AccountID uint64 `json:"account_id"`
}

func (h *LimitHandler) ListAccountLimits(ctx context.Context, req *ListAccountLimitsRequest) ([]*application.LimitDTO, error) {
	limits, err := h.app.ListAccountLimits(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*application.LimitDTO, len(limits))
	for i, l := range limits {
		dtos[i] = application.ToLimitDTO(l)
	}
	return dtos, nil
}

type GetLimitConfigRequest struct {
	AccountID uint64 `json:"account_id"`
}

type LimitConfigDTO struct {
	AccountID         uint64 `json:"account_id"`
	MaxPositionSize   string `json:"max_position_size"`
	MaxOrderSize      string `json:"max_order_size"`
	MaxDailyVolume    string `json:"max_daily_volume"`
	MaxDailyLoss      string `json:"max_daily_loss"`
	MaxExposure       string `json:"max_exposure"`
	MaxLeverage       string `json:"max_leverage"`
	MaxConcentration  string `json:"max_concentration"`
	WarningThreshold  string `json:"warning_threshold"`
	AutoFreezeEnabled bool   `json:"auto_freeze_enabled"`
}

func (h *LimitHandler) GetLimitConfig(ctx context.Context, req *GetLimitConfigRequest) (*LimitConfigDTO, error) {
	config, err := h.app.GetLimitConfig(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, domain.ErrConfigNotFound
	}

	return &LimitConfigDTO{
		AccountID:         config.AccountID,
		MaxPositionSize:   config.MaxPositionSize.String(),
		MaxOrderSize:      config.MaxOrderSize.String(),
		MaxDailyVolume:    config.MaxDailyVolume.String(),
		MaxDailyLoss:      config.MaxDailyLoss.String(),
		MaxExposure:       config.MaxExposure.String(),
		MaxLeverage:       config.MaxLeverage.String(),
		MaxConcentration:  config.MaxConcentration.String(),
		WarningThreshold:  config.WarningThreshold.String(),
		AutoFreezeEnabled: config.AutoFreezeEnabled,
	}, nil
}

type ListBreachesRequest struct {
	AccountID uint64 `json:"account_id"`
	Resolved  bool   `json:"resolved"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

type ListBreachesResponse struct {
	Breaches []*application.BreachDTO `json:"breaches"`
	Total    int64                    `json:"total"`
}

func (h *LimitHandler) ListBreaches(ctx context.Context, req *ListBreachesRequest) (*ListBreachesResponse, error) {
	breaches, total, err := h.app.ListBreaches(ctx, req.AccountID, req.Resolved, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	dtos := make([]*application.BreachDTO, len(breaches))
	for i, b := range breaches {
		dtos[i] = application.ToBreachDTO(b)
	}

	return &ListBreachesResponse{
		Breaches: dtos,
		Total:    total,
	}, nil
}

type ResolveBreachRequest struct {
	BreachID uint64 `json:"breach_id"`
}

func (h *LimitHandler) ResolveBreach(ctx context.Context, req *ResolveBreachRequest) error {
	return h.app.ResolveBreach(ctx, req.BreachID)
}

type FreezeLimitRequest struct {
	LimitID string `json:"limit_id"`
}

func (h *LimitHandler) FreezeLimit(ctx context.Context, req *FreezeLimitRequest) error {
	return h.app.FreezeLimit(ctx, req.LimitID)
}

type UnfreezeLimitRequest struct {
	LimitID string `json:"limit_id"`
}

func (h *LimitHandler) UnfreezeLimit(ctx context.Context, req *UnfreezeLimitRequest) error {
	return h.app.UnfreezeLimit(ctx, req.LimitID)
}

type CreateInstrumentLimitRequest struct {
	AccountID   uint64 `json:"account_id"`
	Symbol      string `json:"symbol"`
	LimitType   string `json:"limit_type"`
	LimitValue  string `json:"limit_value"`
}

func (h *LimitHandler) CreateInstrumentLimit(ctx context.Context, req *CreateInstrumentLimitRequest) error {
	limitValue, _ := decimal.NewFromString(req.LimitValue)
	return h.app.CreateInstrumentLimit(ctx, req.AccountID, req.Symbol, domain.LimitType(req.LimitType), limitValue)
}
