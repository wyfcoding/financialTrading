package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type LimitType string

const (
	LimitTypePosition      LimitType = "POSITION"
	LimitTypeOrder         LimitType = "ORDER"
	LimitTypeDaily         LimitType = "DAILY"
	LimitTypeLoss          LimitType = "LOSS"
	LimitTypeExposure      LimitType = "EXPOSURE"
	LimitTypeMargin        LimitType = "MARGIN"
	LimitTypeLeverage      LimitType = "LEVERAGE"
	LimitTypeConcentration LimitType = "CONCENTRATION"
	LimitTypeTrading       LimitType = "TRADING"
)

type LimitScope string

const (
	ScopeAccount    LimitScope = "ACCOUNT"
	ScopeInstrument LimitScope = "INSTRUMENT"
	ScopeMarket     LimitScope = "MARKET"
	ScopePortfolio  LimitScope = "PORTFOLIO"
)

type LimitStatus string

const (
	LimitStatusActive   LimitStatus = "ACTIVE"
	LimitStatusExceeded LimitStatus = "EXCEEDED"
	LimitStatusFrozen   LimitStatus = "FROZEN"
)

type Limit struct {
	ID               uint64           `json:"id"`
	LimitID          string           `json:"limit_id"`
	AccountID        uint64           `json:"account_id"`
	Type             LimitType        `json:"type"`
	Scope            LimitScope       `json:"scope"`
	Symbol           string           `json:"symbol"`
	CurrentValue     decimal.Decimal  `json:"current_value"`
	LimitValue       decimal.Decimal  `json:"limit_value"`
	WarningThreshold decimal.Decimal  `json:"warning_threshold"`
	UsedPercent      decimal.Decimal  `json:"used_percent"`
	Status           LimitStatus      `json:"status"`
	IsActive         bool             `json:"is_active"`
	StartTime        *time.Time       `json:"start_time"`
	EndTime          *time.Time       `json:"end_time"`
	ResetAt          *time.Time       `json:"reset_at"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

func NewLimit(limitID string, accountID uint64, limitType LimitType, scope LimitScope, limitValue, warningThreshold decimal.Decimal) *Limit {
	return &Limit{
		LimitID:          limitID,
		AccountID:        accountID,
		Type:             limitType,
		Scope:            scope,
		CurrentValue:     decimal.Zero,
		LimitValue:       limitValue,
		WarningThreshold: warningThreshold,
		UsedPercent:      decimal.Zero,
		Status:           LimitStatusActive,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (l *Limit) UpdateCurrentValue(value decimal.Decimal) error {
	l.CurrentValue = value
	if l.LimitValue.IsZero() {
		l.UsedPercent = decimal.Zero
	} else {
		l.UsedPercent = value.Div(l.LimitValue).Mul(decimal.NewFromInt(100))
	}

	if l.UsedPercent.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		l.Status = LimitStatusExceeded
	} else if l.UsedPercent.GreaterThanOrEqual(l.WarningThreshold) {
		l.Status = LimitStatusExceeded
	} else {
		l.Status = LimitStatusActive
	}

	l.UpdatedAt = time.Now()
	return nil
}

func (l *Limit) CheckLimit(additionalValue decimal.Decimal) (*LimitCheckResult, error) {
	newValue := l.CurrentValue.Add(additionalValue)

	if l.LimitValue.IsZero() {
		return &LimitCheckResult{
			Passed:       true,
			CurrentValue: newValue,
			LimitValue:   l.LimitValue,
			UsedPercent:  decimal.Zero,
		}, nil
	}

	usedPercent := newValue.Div(l.LimitValue).Mul(decimal.NewFromInt(100))

	result := &LimitCheckResult{
		CurrentValue: newValue,
		LimitValue:   l.LimitValue,
		UsedPercent:  usedPercent,
	}

	if usedPercent.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		result.Passed = false
		result.Reason = "limit exceeded"
		return result, nil
	}

	if usedPercent.GreaterThanOrEqual(l.WarningThreshold) {
		result.Passed = true
		result.Warning = true
		result.Reason = "approaching limit threshold"
	} else {
		result.Passed = true
	}

	return result, nil
}

func (l *Limit) Freeze() {
	l.Status = LimitStatusFrozen
	l.IsActive = false
	l.UpdatedAt = time.Now()
}

func (l *Limit) Unfreeze() {
	l.Status = LimitStatusActive
	l.IsActive = true
	l.UpdatedAt = time.Now()
}

func (l *Limit) Reset() {
	l.CurrentValue = decimal.Zero
	l.UsedPercent = decimal.Zero
	l.Status = LimitStatusActive
	now := time.Now()
	l.ResetAt = &now
	l.UpdatedAt = now
}

func (l *Limit) SetSymbol(symbol string) {
	l.Symbol = symbol
	l.UpdatedAt = time.Now()
}

func (l *Limit) SetTimeRange(start, end *time.Time) {
	l.StartTime = start
	l.EndTime = end
	l.UpdatedAt = time.Now()
}

type LimitCheckResult struct {
	Passed       bool            `json:"passed"`
	Warning      bool            `json:"warning"`
	CurrentValue decimal.Decimal `json:"current_value"`
	LimitValue   decimal.Decimal `json:"limit_value"`
	UsedPercent  decimal.Decimal `json:"used_percent"`
	Reason       string          `json:"reason"`
}

type LimitBreach struct {
	ID            uint64           `json:"id"`
	BreachID      string           `json:"breach_id"`
	LimitID       string           `json:"limit_id"`
	AccountID     uint64           `json:"account_id"`
	Type          LimitType        `json:"type"`
	CurrentValue  decimal.Decimal  `json:"current_value"`
	LimitValue    decimal.Decimal  `json:"limit_value"`
	BreachPercent decimal.Decimal  `json:"breach_percent"`
	Action        string           `json:"action"`
	ResolvedAt    *time.Time       `json:"resolved_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

func NewLimitBreach(breachID, limitID string, accountID uint64, limitType LimitType, currentValue, limitValue decimal.Decimal) *LimitBreach {
	breachPercent := decimal.Zero
	if !limitValue.IsZero() {
		breachPercent = currentValue.Sub(limitValue).Div(limitValue).Mul(decimal.NewFromInt(100))
	}
	return &LimitBreach{
		BreachID:      breachID,
		LimitID:       limitID,
		AccountID:     accountID,
		Type:          limitType,
		CurrentValue:  currentValue,
		LimitValue:    limitValue,
		BreachPercent: breachPercent,
		Action:        "ALERT",
		CreatedAt:     time.Now(),
	}
}

func (b *LimitBreach) Resolve() {
	now := time.Now()
	b.ResolvedAt = &now
}

func (b *LimitBreach) SetAction(action string) {
	b.Action = action
}

type AccountLimitConfig struct {
	ID                uint64          `json:"id"`
	AccountID         uint64          `json:"account_id"`
	MaxPositionSize   decimal.Decimal `json:"max_position_size"`
	MaxOrderSize      decimal.Decimal `json:"max_order_size"`
	MaxDailyVolume    decimal.Decimal `json:"max_daily_volume"`
	MaxDailyLoss      decimal.Decimal `json:"max_daily_loss"`
	MaxExposure       decimal.Decimal `json:"max_exposure"`
	MaxLeverage       decimal.Decimal `json:"max_leverage"`
	MaxConcentration  decimal.Decimal `json:"max_concentration"`
	WarningThreshold  decimal.Decimal `json:"warning_threshold"`
	AutoFreezeEnabled bool            `json:"auto_freeze_enabled"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func NewAccountLimitConfig(accountID uint64) *AccountLimitConfig {
	return &AccountLimitConfig{
		AccountID:         accountID,
		MaxPositionSize:   decimal.NewFromInt(1000000),
		MaxOrderSize:      decimal.NewFromInt(100000),
		MaxDailyVolume:    decimal.NewFromInt(5000000),
		MaxDailyLoss:      decimal.NewFromInt(50000),
		MaxExposure:       decimal.NewFromInt(2000000),
		MaxLeverage:       decimal.NewFromInt(10),
		MaxConcentration:  decimal.NewFromInt(20),
		WarningThreshold:  decimal.NewFromInt(80),
		AutoFreezeEnabled: true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

var (
	ErrLimitNotFound     = errors.New("limit not found")
	ErrLimitExceeded     = errors.New("limit exceeded")
	ErrConfigNotFound    = errors.New("limit config not found")
	ErrBreachNotFound    = errors.New("breach not found")
	ErrInvalidLimitType  = errors.New("invalid limit type")
	ErrInvalidLimitScope = errors.New("invalid limit scope")
)
