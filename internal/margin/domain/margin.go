package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrMarginAccountNotFound    = errors.New("margin account not found")
	ErrInsufficientMargin       = errors.New("insufficient margin")
	ErrMarginCallRequired       = errors.New("margin call required")
	ErrLiquidationRequired      = errors.New("liquidation required")
	ErrInvalidMarginAmount      = errors.New("invalid margin amount")
	ErrMarginFrozen             = errors.New("margin is frozen")
	ErrPositionNotFound         = errors.New("position not found")
	ErrCollateralNotFound       = errors.New("collateral not found")
	ErrInvalidMarginRatio       = errors.New("invalid margin ratio")
	ErrMarginCallNotFound       = errors.New("margin call not found")
)

type MarginAccountStatus string

const (
	MarginAccountStatusActive     MarginAccountStatus = "ACTIVE"
	MarginAccountStatusFrozen     MarginAccountStatus = "FROZEN"
	MarginAccountStatusRestricted MarginAccountStatus = "RESTRICTED"
	MarginAccountStatusClosed     MarginAccountStatus = "CLOSED"
)

type MarginCallStatus string

const (
	MarginCallStatusPending   MarginCallStatus = "PENDING"
	MarginCallStatusNotified  MarginCallStatus = "NOTIFIED"
	MarginCallStatusMet       MarginCallStatus = "MET"
	MarginCallStatusLiquidated MarginCallStatus = "LIQUIDATED"
	MarginCallStatusExpired   MarginCallStatus = "EXPIRED"
)

type LiquidationStatus string

const (
	LiquidationStatusPending    LiquidationStatus = "PENDING"
	LiquidationStatusProcessing LiquidationStatus = "PROCESSING"
	LiquidationStatusCompleted  LiquidationStatus = "COMPLETED"
	LiquidationStatusPartial    LiquidationStatus = "PARTIAL"
	LiquidationStatusFailed     LiquidationStatus = "FAILED"
)

type MarginType string

const (
	MarginTypeInitial     MarginType = "INITIAL"
	MarginTypeMaintenance MarginType = "MAINTENANCE"
	MarginTypeVariation   MarginType = "VARIATION"
	MarginTypePremium     MarginType = "PREMIUM"
)

type CollateralType string

const (
	CollateralTypeCash       CollateralType = "CASH"
	CollateralTypeSecurities CollateralType = "SECURITIES"
	CollateralTypeBond       CollateralType = "BOND"
	CollateralTypeOther      CollateralType = "OTHER"
)

type MarginAccount struct {
	ID                    string              `json:"id"`
	AccountID             string              `json:"account_id"`
	Currency              string              `json:"currency"`
	TotalEquity           decimal.Decimal     `json:"total_equity"`
	CashBalance           decimal.Decimal     `json:"cash_balance"`
	MarginUsed            decimal.Decimal     `json:"margin_used"`
	MarginAvailable       decimal.Decimal     `json:"margin_available"`
	MaintenanceMargin     decimal.Decimal     `json:"maintenance_margin"`
	InitialMargin         decimal.Decimal     `json:"initial_margin"`
	MarginRatio           decimal.Decimal     `json:"margin_ratio"`
	MarginLevel           decimal.Decimal     `json:"margin_level"`
	UnrealizedPnL         decimal.Decimal     `json:"unrealized_pnl"`
	RealizedPnL           decimal.Decimal     `json:"realized_pnl"`
	TotalCollateralValue  decimal.Decimal     `json:"total_collateral_value"`
	MarginCallThreshold   decimal.Decimal     `json:"margin_call_threshold"`
	LiquidationThreshold  decimal.Decimal     `json:"liquidation_threshold"`
	Status                MarginAccountStatus `json:"status"`
	LastMarginCall        *time.Time          `json:"last_margin_call"`
	LastLiquidation       *time.Time          `json:"last_liquidation"`
	LastCalculation       time.Time           `json:"last_calculation"`
	FrozenAmount          decimal.Decimal     `json:"frozen_amount"`
	ReservedAmount        decimal.Decimal     `json:"reserved_amount"`
	InterestAccrued       decimal.Decimal     `json:"interest_accrued"`
	MarginRate            decimal.Decimal     `json:"margin_rate"`
	LeverageLimit         decimal.Decimal     `json:"leverage_limit"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

func NewMarginAccount(accountID, currency string) *MarginAccount {
	return &MarginAccount{
		AccountID:            accountID,
		Currency:             currency,
		TotalEquity:          decimal.Zero,
		CashBalance:          decimal.Zero,
		MarginUsed:           decimal.Zero,
		MarginAvailable:      decimal.Zero,
		MaintenanceMargin:    decimal.Zero,
		InitialMargin:        decimal.Zero,
		MarginRatio:          decimal.Zero,
		MarginLevel:          decimal.Zero,
		UnrealizedPnL:        decimal.Zero,
		RealizedPnL:          decimal.Zero,
		TotalCollateralValue: decimal.Zero,
		MarginCallThreshold:  decimal.NewFromFloat(0.5),
		LiquidationThreshold: decimal.NewFromFloat(0.25),
		Status:               MarginAccountStatusActive,
		FrozenAmount:         decimal.Zero,
		ReservedAmount:       decimal.Zero,
		InterestAccrued:      decimal.Zero,
		MarginRate:           decimal.NewFromFloat(0.5),
		LeverageLimit:        decimal.NewFromInt(10),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func (ma *MarginAccount) Deposit(amount decimal.Decimal) {
	ma.CashBalance = ma.CashBalance.Add(amount)
	ma.TotalEquity = ma.TotalEquity.Add(amount)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) Withdraw(amount decimal.Decimal) error {
	if ma.CashBalance.LessThan(amount) {
		return ErrInsufficientMargin
	}
	if ma.CashBalance.Sub(amount).LessThan(ma.FrozenAmount.Add(ma.ReservedAmount)) {
		return ErrMarginFrozen
	}
	ma.CashBalance = ma.CashBalance.Sub(amount)
	ma.TotalEquity = ma.TotalEquity.Sub(amount)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
	return nil
}

func (ma *MarginAccount) UseMargin(amount decimal.Decimal) error {
	if ma.MarginAvailable.LessThan(amount) {
		return ErrInsufficientMargin
	}
	ma.MarginUsed = ma.MarginUsed.Add(amount)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
	return nil
}

func (ma *MarginAccount) ReleaseMargin(amount decimal.Decimal) {
	if ma.MarginUsed.GreaterThanOrEqual(amount) {
		ma.MarginUsed = ma.MarginUsed.Sub(amount)
	} else {
		ma.MarginUsed = decimal.Zero
	}
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) UpdatePnL(unrealizedPnL decimal.Decimal) {
	ma.UnrealizedPnL = unrealizedPnL
	ma.TotalEquity = ma.CashBalance.Add(ma.TotalCollateralValue).Add(unrealizedPnL)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) AddRealizedPnL(amount decimal.Decimal) {
	ma.RealizedPnL = ma.RealizedPnL.Add(amount)
	ma.CashBalance = ma.CashBalance.Add(amount)
	ma.TotalEquity = ma.TotalEquity.Add(amount)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) Freeze(amount decimal.Decimal) error {
	if ma.CashBalance.Sub(ma.FrozenAmount).LessThan(amount) {
		return ErrInsufficientMargin
	}
	ma.FrozenAmount = ma.FrozenAmount.Add(amount)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
	return nil
}

func (ma *MarginAccount) Unfreeze(amount decimal.Decimal) {
	if ma.FrozenAmount.GreaterThanOrEqual(amount) {
		ma.FrozenAmount = ma.FrozenAmount.Sub(amount)
	} else {
		ma.FrozenAmount = decimal.Zero
	}
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) Reserve(amount decimal.Decimal) error {
	available := ma.CashBalance.Sub(ma.FrozenAmount).Sub(ma.ReservedAmount)
	if available.LessThan(amount) {
		return ErrInsufficientMargin
	}
	ma.ReservedAmount = ma.ReservedAmount.Add(amount)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
	return nil
}

func (ma *MarginAccount) ReleaseReserved(amount decimal.Decimal) {
	if ma.ReservedAmount.GreaterThanOrEqual(amount) {
		ma.ReservedAmount = ma.ReservedAmount.Sub(amount)
	} else {
		ma.ReservedAmount = decimal.Zero
	}
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) recalculateMargin() {
	ma.MarginAvailable = ma.TotalEquity.Sub(ma.MarginUsed).Sub(ma.FrozenAmount).Sub(ma.ReservedAmount)
	if ma.MarginAvailable.LessThan(decimal.Zero) {
		ma.MarginAvailable = decimal.Zero
	}
	if ma.MaintenanceMargin.GreaterThan(decimal.Zero) {
		ma.MarginRatio = ma.TotalEquity.Div(ma.MaintenanceMargin)
		ma.MarginLevel = ma.TotalEquity.Sub(ma.MaintenanceMargin).Div(ma.TotalEquity).Mul(decimal.NewFromInt(100))
	}
}

func (ma *MarginAccount) IsMarginCallRequired() bool {
	return ma.MarginRatio.LessThanOrEqual(ma.MarginCallThreshold)
}

func (ma *MarginAccount) IsLiquidationRequired() bool {
	return ma.MarginRatio.LessThanOrEqual(ma.LiquidationThreshold)
}

func (ma *MarginAccount) FreezeAccount() {
	ma.Status = MarginAccountStatusFrozen
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) UnfreezeAccount() {
	ma.Status = MarginAccountStatusActive
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) RestrictAccount() {
	ma.Status = MarginAccountStatusRestricted
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) CloseAccount() {
	ma.Status = MarginAccountStatusClosed
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) AddCollateral(value decimal.Decimal) {
	ma.TotalCollateralValue = ma.TotalCollateralValue.Add(value)
	ma.TotalEquity = ma.TotalEquity.Add(value)
	ma.recalculateMargin()
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) RemoveCollateral(value decimal.Decimal) {
	if ma.TotalCollateralValue.GreaterThanOrEqual(value) {
		ma.TotalCollateralValue = ma.TotalCollateralValue.Sub(value)
		ma.TotalEquity = ma.TotalEquity.Sub(value)
		ma.recalculateMargin()
	}
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) AccrueInterest(amount decimal.Decimal) {
	ma.InterestAccrued = ma.InterestAccrued.Add(amount)
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) PayInterest(amount decimal.Decimal) {
	if ma.InterestAccrued.GreaterThanOrEqual(amount) {
		ma.InterestAccrued = ma.InterestAccrued.Sub(amount)
		ma.CashBalance = ma.CashBalance.Sub(amount)
		ma.TotalEquity = ma.TotalEquity.Sub(amount)
		ma.recalculateMargin()
	}
	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) MarkCalculated() {
	ma.LastCalculation = time.Now()
	ma.UpdatedAt = time.Now()
}

type MarginPosition struct {
	ID                string          `json:"id"`
	AccountID         string          `json:"account_id"`
	Symbol            string          `json:"symbol"`
	PositionType      string          `json:"position_type"`
	Quantity          decimal.Decimal `json:"quantity"`
	EntryPrice        decimal.Decimal `json:"entry_price"`
	CurrentPrice      decimal.Decimal `json:"current_price"`
	MarketValue       decimal.Decimal `json:"market_value"`
	InitialMargin     decimal.Decimal `json:"initial_margin"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"`
	UnrealizedPnL     decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL       decimal.Decimal `json:"realized_pnl"`
	MarginRate        decimal.Decimal `json:"margin_rate"`
	Leverage          decimal.Decimal `json:"leverage"`
	OpenedAt          time.Time       `json:"opened_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	ClosedAt          *time.Time      `json:"closed_at"`
	Status            string          `json:"status"`
}

func NewMarginPosition(accountID, symbol, positionType string, quantity, entryPrice, marginRate decimal.Decimal) *MarginPosition {
	marketValue := quantity.Mul(entryPrice)
	initialMargin := marketValue.Mul(marginRate)
	maintenanceMargin := initialMargin.Mul(decimal.NewFromFloat(0.5))
	
	return &MarginPosition{
		AccountID:         accountID,
		Symbol:            symbol,
		PositionType:      positionType,
		Quantity:          quantity,
		EntryPrice:        entryPrice,
		CurrentPrice:      entryPrice,
		MarketValue:       marketValue,
		InitialMargin:     initialMargin,
		MaintenanceMargin: maintenanceMargin,
		UnrealizedPnL:     decimal.Zero,
		RealizedPnL:       decimal.Zero,
		MarginRate:        marginRate,
		Leverage:          decimal.NewFromInt(1).Div(marginRate),
		OpenedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		Status:            "OPEN",
	}
}

func (mp *MarginPosition) UpdatePrice(currentPrice decimal.Decimal) {
	mp.CurrentPrice = currentPrice
	mp.MarketValue = mp.Quantity.Mul(currentPrice)
	mp.UnrealizedPnL = mp.Quantity.Mul(currentPrice.Sub(mp.EntryPrice))
	mp.UpdatedAt = time.Now()
}

func (mp *MarginPosition) AddQuantity(quantity, price decimal.Decimal) {
	totalCost := mp.Quantity.Mul(mp.EntryPrice).Add(quantity.Mul(price))
	mp.Quantity = mp.Quantity.Add(quantity)
	mp.EntryPrice = totalCost.Div(mp.Quantity)
	mp.MarketValue = mp.Quantity.Mul(mp.CurrentPrice)
	mp.InitialMargin = mp.MarketValue.Mul(mp.MarginRate)
	mp.MaintenanceMargin = mp.InitialMargin.Mul(decimal.NewFromFloat(0.5))
	mp.UpdatedAt = time.Now()
}

func (mp *MarginPosition) ReduceQuantity(quantity decimal.Decimal) decimal.Decimal {
	if mp.Quantity.LessThanOrEqual(quantity) {
		pnl := mp.UnrealizedPnL
		mp.Quantity = decimal.Zero
		mp.MarketValue = decimal.Zero
		mp.InitialMargin = decimal.Zero
		mp.MaintenanceMargin = decimal.Zero
		mp.UnrealizedPnL = decimal.Zero
		now := time.Now()
		mp.ClosedAt = &now
		mp.Status = "CLOSED"
		mp.UpdatedAt = now
		return pnl
	}
	
	realizedPnL := quantity.Mul(mp.CurrentPrice.Sub(mp.EntryPrice))
	mp.Quantity = mp.Quantity.Sub(quantity)
	mp.MarketValue = mp.Quantity.Mul(mp.CurrentPrice)
	mp.InitialMargin = mp.MarketValue.Mul(mp.MarginRate)
	mp.MaintenanceMargin = mp.InitialMargin.Mul(decimal.NewFromFloat(0.5))
	mp.UnrealizedPnL = mp.Quantity.Mul(mp.CurrentPrice.Sub(mp.EntryPrice))
	mp.RealizedPnL = mp.RealizedPnL.Add(realizedPnL)
	mp.UpdatedAt = time.Now()
	return realizedPnL
}

func (mp *MarginPosition) ClosePosition(closePrice decimal.Decimal) decimal.Decimal {
	mp.CurrentPrice = closePrice
	mp.UnrealizedPnL = mp.Quantity.Mul(closePrice.Sub(mp.EntryPrice))
	realizedPnL := mp.UnrealizedPnL
	mp.RealizedPnL = mp.RealizedPnL.Add(realizedPnL)
	mp.Quantity = decimal.Zero
	mp.MarketValue = decimal.Zero
	mp.InitialMargin = decimal.Zero
	mp.MaintenanceMargin = decimal.Zero
	mp.UnrealizedPnL = decimal.Zero
	now := time.Now()
	mp.ClosedAt = &now
	mp.Status = "CLOSED"
	mp.UpdatedAt = now
	return realizedPnL
}

type MarginCall struct {
	ID               string           `json:"id"`
	AccountID        string           `json:"account_id"`
	MarginRatio      decimal.Decimal  `json:"margin_ratio"`
	RequiredMargin   decimal.Decimal  `json:"required_margin"`
	DeficiencyAmount decimal.Decimal  `json:"deficiency_amount"`
	Status           MarginCallStatus `json:"status"`
	Deadline         time.Time        `json:"deadline"`
	NotifiedAt       *time.Time       `json:"notified_at"`
	MetAt            *time.Time       `json:"met_at"`
	LiquidatedAt     *time.Time       `json:"liquidated_at"`
	Positions        []string         `json:"positions"`
	Actions          []MarginAction   `json:"actions"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type MarginAction struct {
	ActionType  string          `json:"action_type"`
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
	TakenAt     time.Time       `json:"taken_at"`
}

func NewMarginCall(accountID string, marginRatio, requiredMargin, deficiencyAmount decimal.Decimal, deadlineDuration time.Duration) *MarginCall {
	return &MarginCall{
		AccountID:        accountID,
		MarginRatio:      marginRatio,
		RequiredMargin:   requiredMargin,
		DeficiencyAmount: deficiencyAmount,
		Status:           MarginCallStatusPending,
		Deadline:         time.Now().Add(deadlineDuration),
		Positions:        []string{},
		Actions:          []MarginAction{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (mc *MarginCall) Notify() {
	now := time.Now()
	mc.Status = MarginCallStatusNotified
	mc.NotifiedAt = &now
	mc.UpdatedAt = now
}

func (mc *MarginCall) Meet() {
	now := time.Now()
	mc.Status = MarginCallStatusMet
	mc.MetAt = &now
	mc.UpdatedAt = now
}

func (mc *MarginCall) Liquidate() {
	now := time.Now()
	mc.Status = MarginCallStatusLiquidated
	mc.LiquidatedAt = &now
	mc.UpdatedAt = now
}

func (mc *MarginCall) Expire() {
	now := time.Now()
	mc.Status = MarginCallStatusExpired
	mc.UpdatedAt = now
}

func (mc *MarginCall) IsExpired() bool {
	return time.Now().After(mc.Deadline)
}

func (mc *MarginCall) AddAction(actionType, description string, amount decimal.Decimal) {
	action := MarginAction{
		ActionType:  actionType,
		Amount:      amount,
		Description: description,
		TakenAt:     time.Now(),
	}
	mc.Actions = append(mc.Actions, action)
	mc.UpdatedAt = time.Now()
}

type Liquidation struct {
	ID              string            `json:"id"`
	AccountID       string            `json:"account_id"`
	MarginCallID    string            `json:"margin_call_id"`
	Positions       []LiquidationPosition `json:"positions"`
	TotalValue      decimal.Decimal   `json:"total_value"`
	TotalRecovered  decimal.Decimal   `json:"total_recovered"`
	Deficiency      decimal.Decimal   `json:"deficiency"`
	Status          LiquidationStatus `json:"status"`
	TriggeredAt     time.Time         `json:"triggered_at"`
	StartedAt       *time.Time        `json:"started_at"`
	CompletedAt     *time.Time        `json:"completed_at"`
	Reason          string            `json:"reason"`
	Notes           string            `json:"notes"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type LiquidationPosition struct {
	PositionID    string          `json:"position_id"`
	Symbol        string          `json:"symbol"`
	Quantity      decimal.Decimal `json:"quantity"`
	LiquidationPrice decimal.Decimal `json:"liquidation_price"`
	MarketValue   decimal.Decimal `json:"market_value"`
	Recovered     decimal.Decimal `json:"recovered"`
	Status        string          `json:"status"`
}

func NewLiquidation(accountID, marginCallID, reason string) *Liquidation {
	return &Liquidation{
		AccountID:     accountID,
		MarginCallID:  marginCallID,
		Positions:     []LiquidationPosition{},
		TotalValue:    decimal.Zero,
		TotalRecovered: decimal.Zero,
		Deficiency:    decimal.Zero,
		Status:        LiquidationStatusPending,
		TriggeredAt:   time.Now(),
		Reason:        reason,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (l *Liquidation) Start() {
	now := time.Now()
	l.Status = LiquidationStatusProcessing
	l.StartedAt = &now
	l.UpdatedAt = now
}

func (l *Liquidation) AddPosition(pos LiquidationPosition) {
	l.Positions = append(l.Positions, pos)
	l.TotalValue = l.TotalValue.Add(pos.MarketValue)
	l.UpdatedAt = time.Now()
}

func (l *Liquidation) UpdateRecovered(positionID string, recovered decimal.Decimal) {
	for i, pos := range l.Positions {
		if pos.PositionID == positionID {
			l.Positions[i].Recovered = recovered
			l.Positions[i].Status = "LIQUIDATED"
			l.TotalRecovered = l.TotalRecovered.Add(recovered)
			break
		}
	}
	l.UpdatedAt = time.Now()
}

func (l *Liquidation) Complete() {
	now := time.Now()
	l.Status = LiquidationStatusCompleted
	l.CompletedAt = &now
	l.Deficiency = l.TotalValue.Sub(l.TotalRecovered)
	if l.Deficiency.LessThan(decimal.Zero) {
		l.Deficiency = decimal.Zero
	}
	l.UpdatedAt = now
}

func (l *Liquidation) PartialComplete() {
	now := time.Now()
	l.Status = LiquidationStatusPartial
	l.CompletedAt = &now
	l.Deficiency = l.TotalValue.Sub(l.TotalRecovered)
	l.UpdatedAt = now
}

func (l *Liquidation) Fail(reason string) {
	now := time.Now()
	l.Status = LiquidationStatusFailed
	l.CompletedAt = &now
	l.Notes = reason
	l.UpdatedAt = now
}

type Collateral struct {
	ID              string          `json:"id"`
	AccountID       string          `json:"account_id"`
	Type            CollateralType  `json:"type"`
	Symbol          string          `json:"symbol"`
	Quantity        decimal.Decimal `json:"quantity"`
	OriginalValue   decimal.Decimal `json:"original_value"`
	CurrentValue    decimal.Decimal `json:"current_value"`
	Haircut         decimal.Decimal `json:"haircut"`
	EligibleValue   decimal.Decimal `json:"eligible_value"`
	Status          string          `json:"status"`
	Locked          bool            `json:"locked"`
	LockedAt        *time.Time      `json:"locked_at"`
	LastValuation   time.Time       `json:"last_valuation"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func NewCollateral(accountID string, collateralType CollateralType, symbol string, quantity, value, haircut decimal.Decimal) *Collateral {
	return &Collateral{
		AccountID:     accountID,
		Type:          collateralType,
		Symbol:        symbol,
		Quantity:      quantity,
		OriginalValue: value,
		CurrentValue:  value,
		Haircut:       haircut,
		EligibleValue: value.Mul(decimal.NewFromInt(1).Sub(haircut)),
		Status:        "ACTIVE",
		Locked:        false,
		LastValuation: time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (c *Collateral) UpdateValue(newValue decimal.Decimal) {
	c.CurrentValue = newValue
	c.EligibleValue = newValue.Mul(decimal.NewFromInt(1).Sub(c.Haircut))
	c.LastValuation = time.Now()
	c.UpdatedAt = time.Now()
}

func (c *Collateral) Lock() {
	now := time.Now()
	c.Locked = true
	c.LockedAt = &now
	c.UpdatedAt = now
}

func (c *Collateral) Unlock() {
	c.Locked = false
	c.LockedAt = nil
	c.UpdatedAt = time.Now()
}

func (c *Collateral) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

type MarginRequirement struct {
	ID                string          `json:"id"`
	AccountID         string          `json:"account_id"`
	Symbol            string          `json:"symbol"`
	PositionType      string          `json:"position_type"`
	Quantity          decimal.Decimal `json:"quantity"`
	Price             decimal.Decimal `json:"price"`
	MarketValue       decimal.Decimal `json:"market_value"`
	InitialMarginRate decimal.Decimal `json:"initial_margin_rate"`
	InitialMargin     decimal.Decimal `json:"initial_margin"`
	MaintenanceRate   decimal.Decimal `json:"maintenance_rate"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"`
	MarginType        MarginType      `json:"margin_type"`
	CalculatedAt      time.Time       `json:"calculated_at"`
}

func NewMarginRequirement(accountID, symbol, positionType string, quantity, price, initialRate, maintenanceRate decimal.Decimal) *MarginRequirement {
	marketValue := quantity.Mul(price)
	return &MarginRequirement{
		AccountID:         accountID,
		Symbol:            symbol,
		PositionType:      positionType,
		Quantity:          quantity,
		Price:             price,
		MarketValue:       marketValue,
		InitialMarginRate: initialRate,
		InitialMargin:     marketValue.Mul(initialRate),
		MaintenanceRate:   maintenanceRate,
		MaintenanceMargin: marketValue.Mul(maintenanceRate),
		MarginType:        MarginTypeInitial,
		CalculatedAt:      time.Now(),
	}
}

type MarginDeposit struct {
	ID            string          `json:"id"`
	AccountID     string          `json:"account_id"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	SourceAccount string          `json:"source_account"`
	ReferenceID   string          `json:"reference_id"`
	ProcessedAt   *time.Time      `json:"processed_at"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func NewMarginDeposit(accountID string, amount decimal.Decimal, currency, depositType string) *MarginDeposit {
	return &MarginDeposit{
		AccountID: accountID,
		Amount:    amount,
		Currency:  currency,
		Type:      depositType,
		Status:    "PENDING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (md *MarginDeposit) Process() {
	now := time.Now()
	md.Status = "COMPLETED"
	md.ProcessedAt = &now
	md.UpdatedAt = now
}

func (md *MarginDeposit) Fail(reason string) {
	md.Status = "FAILED"
	md.UpdatedAt = time.Now()
}

type MarginWithdrawal struct {
	ID             string          `json:"id"`
	AccountID      string          `json:"account_id"`
	Amount         decimal.Decimal `json:"amount"`
	Currency       string          `json:"currency"`
	Status         string          `json:"status"`
	DestinationAccount string      `json:"destination_account"`
	ReferenceID    string          `json:"reference_id"`
	ProcessedAt    *time.Time      `json:"processed_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func NewMarginWithdrawal(accountID string, amount decimal.Decimal, currency string) *MarginWithdrawal {
	return &MarginWithdrawal{
		AccountID: accountID,
		Amount:    amount,
		Currency:  currency,
		Status:    "PENDING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (mw *MarginWithdrawal) Process() {
	now := time.Now()
	mw.Status = "COMPLETED"
	mw.ProcessedAt = &now
	mw.UpdatedAt = now
}

func (mw *MarginWithdrawal) Fail(reason string) {
	mw.Status = "FAILED"
	mw.UpdatedAt = time.Now()
}

type MarginAlert struct {
	ID          string          `json:"id"`
	AccountID   string          `json:"account_id"`
	AlertType   string          `json:"alert_type"`
	Severity    string          `json:"severity"`
	Message     string          `json:"message"`
	MarginRatio decimal.Decimal `json:"margin_ratio"`
	Threshold   decimal.Decimal `json:"threshold"`
	Read        bool            `json:"read"`
	ReadAt      *time.Time      `json:"read_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

func NewMarginAlert(accountID, alertType, severity, message string, marginRatio, threshold decimal.Decimal) *MarginAlert {
	return &MarginAlert{
		AccountID:   accountID,
		AlertType:   alertType,
		Severity:    severity,
		Message:     message,
		MarginRatio: marginRatio,
		Threshold:   threshold,
		Read:        false,
		CreatedAt:   time.Now(),
	}
}

func (ma *MarginAlert) MarkRead() {
	now := time.Now()
	ma.Read = true
	ma.ReadAt = &now
}

type MarginAccountRepository interface {
	Create(ctx context.Context, account *MarginAccount) error
	Update(ctx context.Context, account *MarginAccount) error
	FindByID(ctx context.Context, id string) (*MarginAccount, error)
	FindByAccountID(ctx context.Context, accountID string) (*MarginAccount, error)
	FindByStatus(ctx context.Context, status MarginAccountStatus, limit, offset int) ([]*MarginAccount, int64, error)
	FindRequiringMarginCall(ctx context.Context, threshold decimal.Decimal) ([]*MarginAccount, error)
	FindRequiringLiquidation(ctx context.Context, threshold decimal.Decimal) ([]*MarginAccount, error)
	Delete(ctx context.Context, id string) error
}

type MarginPositionRepository interface {
	Create(ctx context.Context, position *MarginPosition) error
	Update(ctx context.Context, position *MarginPosition) error
	FindByID(ctx context.Context, id string) (*MarginPosition, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*MarginPosition, error)
	FindBySymbol(ctx context.Context, accountID, symbol string) ([]*MarginPosition, error)
	FindOpenPositions(ctx context.Context, accountID string) ([]*MarginPosition, error)
	Delete(ctx context.Context, id string) error
}

type MarginCallRepository interface {
	Create(ctx context.Context, marginCall *MarginCall) error
	Update(ctx context.Context, marginCall *MarginCall) error
	FindByID(ctx context.Context, id string) (*MarginCall, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*MarginCall, error)
	FindPending(ctx context.Context, limit, offset int) ([]*MarginCall, int64, error)
	FindExpired(ctx context.Context) ([]*MarginCall, error)
	Delete(ctx context.Context, id string) error
}

type LiquidationRepository interface {
	Create(ctx context.Context, liquidation *Liquidation) error
	Update(ctx context.Context, liquidation *Liquidation) error
	FindByID(ctx context.Context, id string) (*Liquidation, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*Liquidation, error)
	FindPending(ctx context.Context, limit int) ([]*Liquidation, error)
	Delete(ctx context.Context, id string) error
}

type CollateralRepository interface {
	Create(ctx context.Context, collateral *Collateral) error
	Update(ctx context.Context, collateral *Collateral) error
	FindByID(ctx context.Context, id string) (*Collateral, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*Collateral, error)
	FindBySymbol(ctx context.Context, accountID, symbol string) ([]*Collateral, error)
	Delete(ctx context.Context, id string) error
}

type MarginRequirementRepository interface {
	Create(ctx context.Context, requirement *MarginRequirement) error
	FindByID(ctx context.Context, id string) (*MarginRequirement, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*MarginRequirement, error)
	Delete(ctx context.Context, id string) error
}

type MarginDepositRepository interface {
	Create(ctx context.Context, deposit *MarginDeposit) error
	FindByID(ctx context.Context, id string) (*MarginDeposit, error)
	FindByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*MarginDeposit, int64, error)
	Delete(ctx context.Context, id string) error
}

type MarginWithdrawalRepository interface {
	Create(ctx context.Context, withdrawal *MarginWithdrawal) error
	FindByID(ctx context.Context, id string) (*MarginWithdrawal, error)
	FindByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*MarginWithdrawal, int64, error)
	Delete(ctx context.Context, id string) error
}

type MarginAlertRepository interface {
	Create(ctx context.Context, alert *MarginAlert) error
	FindByID(ctx context.Context, id string) (*MarginAlert, error)
	FindByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*MarginAlert, int64, error)
	FindUnread(ctx context.Context, accountID string) ([]*MarginAlert, error)
	Delete(ctx context.Context, id string) error
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
