package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrFinancingAccountNotFound    = errors.New("financing account not found")
	ErrInsufficientCredit          = errors.New("insufficient credit")
	ErrCollateralInsufficient      = errors.New("collateral insufficient")
	ErrFinancingNotFound           = errors.New("financing not found")
	ErrStockBorrowNotFound         = errors.New("stock borrow not found")
	ErrRepaymentFailed             = errors.New("repayment failed")
	ErrMaintenanceRatioTooLow      = errors.New("maintenance ratio too low")
	ErrStockUnavailable            = errors.New("stock unavailable for borrowing")
	ErrInvalidFinancingAmount      = errors.New("invalid financing amount")
	ErrFinancingAccountFrozen      = errors.New("financing account frozen")
	ErrCollateralNotFound          = errors.New("collateral not found")
	ErrInterestCalculationFailed   = errors.New("interest calculation failed")
)

type FinancingAccountStatus string

const (
	FinancingAccountStatusActive     FinancingAccountStatus = "ACTIVE"
	FinancingAccountStatusFrozen     FinancingAccountStatus = "FROZEN"
	FinancingAccountStatusRestricted FinancingAccountStatus = "RESTRICTED"
	FinancingAccountStatusClosed     FinancingAccountStatus = "CLOSED"
)

type FinancingType string

const (
	FinancingTypeMargin    FinancingType = "MARGIN"
	FinancingTypeCash      FinancingType = "CASH"
	FinancingTypeSecurities FinancingType = "SECURITIES"
)

type BorrowingStatus string

const (
	BorrowingStatusActive    BorrowingStatus = "ACTIVE"
	BorrowingStatusRepaid    BorrowingStatus = "REPAID"
	BorrowingStatusDefaulted BorrowingStatus = "DEFAULTED"
	BorrowingStatusExtended  BorrowingStatus = "EXTENDED"
)

type CollateralStatus string

const (
	CollateralStatusLocked    CollateralStatus = "LOCKED"
	CollateralStatusReleased  CollateralStatus = "RELEASED"
	CollateralStatusLiquidated CollateralStatus = "LIQUIDATED"
)

type FinancingAccount struct {
	ID                     string                `json:"id"`
	AccountID              string                `json:"account_id"`
	Currency               string                `json:"currency"`
	TotalCreditLimit       decimal.Decimal       `json:"total_credit_limit"`
	UsedCredit             decimal.Decimal       `json:"used_credit"`
	AvailableCredit        decimal.Decimal       `json:"available_credit"`
	TotalCollateralValue   decimal.Decimal       `json:"total_collateral_value"`
	AdjustableCollateral   decimal.Decimal       `json:"adjustable_collateral"`
	MaintenanceRatio       decimal.Decimal       `json:"maintenance_ratio"`
	MaintenanceRatioLimit  decimal.Decimal       `json:"maintenance_ratio_limit"`
	FinancingBalance       decimal.Decimal       `json:"financing_balance"`
	FinancingInterest      decimal.Decimal       `json:"financing_interest"`
	StockBorrowValue       decimal.Decimal       `json:"stock_borrow_value"`
	StockBorrowInterest    decimal.Decimal       `json:"stock_borrow_interest"`
	TotalInterestAccrued   decimal.Decimal       `json:"total_interest_accrued"`
	UnrealizedPnL          decimal.Decimal       `json:"unrealized_pnl"`
	RealizedPnL            decimal.Decimal       `json:"realized_pnl"`
	TotalEquity            decimal.Decimal       `json:"total_equity"`
	Status                 FinancingAccountStatus `json:"status"`
	RiskLevel              string                `json:"risk_level"`
	FinancingRate          decimal.Decimal       `json:"financing_rate"`
	StockBorrowRate        decimal.Decimal       `json:"stock_borrow_rate"`
	LastInterestCalc       time.Time             `json:"last_interest_calc"`
	LastMaintenanceCheck   time.Time             `json:"last_maintenance_check"`
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`
}

func NewFinancingAccount(accountID, currency string, creditLimit decimal.Decimal) *FinancingAccount {
	return &FinancingAccount{
		AccountID:             accountID,
		Currency:              currency,
		TotalCreditLimit:      creditLimit,
		UsedCredit:            decimal.Zero,
		AvailableCredit:       creditLimit,
		TotalCollateralValue:  decimal.Zero,
		AdjustableCollateral:  decimal.Zero,
		MaintenanceRatio:      decimal.Zero,
		MaintenanceRatioLimit: decimal.NewFromFloat(1.3),
		FinancingBalance:      decimal.Zero,
		FinancingInterest:     decimal.Zero,
		StockBorrowValue:      decimal.Zero,
		StockBorrowInterest:   decimal.Zero,
		TotalInterestAccrued:  decimal.Zero,
		UnrealizedPnL:         decimal.Zero,
		RealizedPnL:           decimal.Zero,
		TotalEquity:           decimal.Zero,
		Status:                FinancingAccountStatusActive,
		RiskLevel:             "LOW",
		FinancingRate:         decimal.NewFromFloat(0.06),
		StockBorrowRate:       decimal.NewFromFloat(0.08),
		LastInterestCalc:      time.Now(),
		LastMaintenanceCheck:  time.Now(),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

func (fa *FinancingAccount) BorrowCash(amount decimal.Decimal) error {
	if fa.Status != FinancingAccountStatusActive {
		return ErrFinancingAccountFrozen
	}
	if fa.AvailableCredit.LessThan(amount) {
		return ErrInsufficientCredit
	}
	fa.UsedCredit = fa.UsedCredit.Add(amount)
	fa.AvailableCredit = fa.AvailableCredit.Sub(amount)
	fa.FinancingBalance = fa.FinancingBalance.Add(amount)
	fa.recalculateMaintenanceRatio()
	fa.UpdatedAt = time.Now()
	return nil
}

func (fa *FinancingAccount) RepayCash(amount decimal.Decimal) error {
	if fa.FinancingBalance.LessThan(amount) {
		amount = fa.FinancingBalance
	}
	fa.FinancingBalance = fa.FinancingBalance.Sub(amount)
	fa.UsedCredit = fa.UsedCredit.Sub(amount)
	fa.AvailableCredit = fa.AvailableCredit.Add(amount)
	fa.recalculateMaintenanceRatio()
	fa.UpdatedAt = time.Now()
	return nil
}

func (fa *FinancingAccount) BorrowStock(value decimal.Decimal) error {
	if fa.Status != FinancingAccountStatusActive {
		return ErrFinancingAccountFrozen
	}
	if fa.AvailableCredit.LessThan(value) {
		return ErrInsufficientCredit
	}
	fa.UsedCredit = fa.UsedCredit.Add(value)
	fa.AvailableCredit = fa.AvailableCredit.Sub(value)
	fa.StockBorrowValue = fa.StockBorrowValue.Add(value)
	fa.recalculateMaintenanceRatio()
	fa.UpdatedAt = time.Now()
	return nil
}

func (fa *FinancingAccount) ReturnStock(value decimal.Decimal) error {
	if fa.StockBorrowValue.LessThan(value) {
		value = fa.StockBorrowValue
	}
	fa.StockBorrowValue = fa.StockBorrowValue.Sub(value)
	fa.UsedCredit = fa.UsedCredit.Sub(value)
	fa.AvailableCredit = fa.AvailableCredit.Add(value)
	fa.recalculateMaintenanceRatio()
	fa.UpdatedAt = time.Now()
	return nil
}

func (fa *FinancingAccount) AddCollateral(value decimal.Decimal) {
	fa.TotalCollateralValue = fa.TotalCollateralValue.Add(value)
	fa.recalculateMaintenanceRatio()
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) RemoveCollateral(value decimal.Decimal) error {
	if fa.TotalCollateralValue.LessThan(value) {
		return ErrCollateralInsufficient
	}
	fa.TotalCollateralValue = fa.TotalCollateralValue.Sub(value)
	fa.recalculateMaintenanceRatio()
	if fa.MaintenanceRatio.LessThan(fa.MaintenanceRatioLimit) {
		return ErrMaintenanceRatioTooLow
	}
	fa.UpdatedAt = time.Now()
	return nil
}

func (fa *FinancingAccount) AccrueFinancingInterest(amount decimal.Decimal) {
	fa.FinancingInterest = fa.FinancingInterest.Add(amount)
	fa.TotalInterestAccrued = fa.TotalInterestAccrued.Add(amount)
	fa.LastInterestCalc = time.Now()
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) AccrueStockBorrowInterest(amount decimal.Decimal) {
	fa.StockBorrowInterest = fa.StockBorrowInterest.Add(amount)
	fa.TotalInterestAccrued = fa.TotalInterestAccrued.Add(amount)
	fa.LastInterestCalc = time.Now()
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) PayInterest(amount decimal.Decimal) {
	if fa.TotalInterestAccrued.LessThan(amount) {
		amount = fa.TotalInterestAccrued
	}
	fa.TotalInterestAccrued = fa.TotalInterestAccrued.Sub(amount)
	if fa.FinancingInterest.GreaterThanOrEqual(amount) {
		fa.FinancingInterest = fa.FinancingInterest.Sub(amount)
	} else {
		remaining := amount.Sub(fa.FinancingInterest)
		fa.FinancingInterest = decimal.Zero
		fa.StockBorrowInterest = fa.StockBorrowInterest.Sub(remaining)
	}
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) UpdatePnL(unrealizedPnL decimal.Decimal) {
	fa.UnrealizedPnL = unrealizedPnL
	fa.TotalEquity = fa.TotalCollateralValue.Add(unrealizedPnL)
	fa.recalculateMaintenanceRatio()
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) recalculateMaintenanceRatio() {
	totalDebt := fa.FinancingBalance.Add(fa.FinancingInterest).Add(fa.StockBorrowValue).Add(fa.StockBorrowInterest)
	if totalDebt.GreaterThan(decimal.Zero) {
		fa.MaintenanceRatio = fa.TotalCollateralValue.Div(totalDebt)
	} else {
		fa.MaintenanceRatio = decimal.NewFromInt(999)
	}
}

func (fa *FinancingAccount) IsMarginCallRequired() bool {
	return fa.MaintenanceRatio.LessThan(fa.MaintenanceRatioLimit)
}

func (fa *FinancingAccount) Freeze() {
	fa.Status = FinancingAccountStatusFrozen
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) Unfreeze() {
	fa.Status = FinancingAccountStatusActive
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) Restrict() {
	fa.Status = FinancingAccountStatusRestricted
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) Close() {
	fa.Status = FinancingAccountStatusClosed
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) UpdateRiskLevel(level string) {
	fa.RiskLevel = level
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingAccount) MarkMaintenanceCheck() {
	fa.LastMaintenanceCheck = time.Now()
	fa.UpdatedAt = time.Now()
}

type FinancingRecord struct {
	ID              string          `json:"id"`
	AccountID       string          `json:"account_id"`
	FinancingType   FinancingType   `json:"financing_type"`
	Amount          decimal.Decimal `json:"amount"`
	InterestRate    decimal.Decimal `json:"interest_rate"`
	InterestAccrued decimal.Decimal `json:"interest_accrued"`
	RepaidAmount    decimal.Decimal `json:"repaid_amount"`
	RepaidInterest  decimal.Decimal `json:"repaid_interest"`
	Outstanding     decimal.Decimal `json:"outstanding"`
	Status          BorrowingStatus `json:"status"`
	BorrowedAt      time.Time       `json:"borrowed_at"`
	DueDate         time.Time       `json:"due_date"`
	LastInterestCalc time.Time      `json:"last_interest_calc"`
	RepaidAt        *time.Time      `json:"repaid_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func NewFinancingRecord(accountID string, financingType FinancingType, amount, interestRate decimal.Decimal, termDays int) *FinancingRecord {
	now := time.Now()
	return &FinancingRecord{
		AccountID:       accountID,
		FinancingType:   financingType,
		Amount:          amount,
		InterestRate:    interestRate,
		InterestAccrued: decimal.Zero,
		RepaidAmount:    decimal.Zero,
		RepaidInterest:  decimal.Zero,
		Outstanding:     amount,
		Status:          BorrowingStatusActive,
		BorrowedAt:      now,
		DueDate:         now.AddDate(0, 0, termDays),
		LastInterestCalc: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (fr *FinancingRecord) AccrueInterest(days int) {
	dailyRate := fr.InterestRate.Div(decimal.NewFromInt(365))
	interest := fr.Outstanding.Mul(dailyRate).Mul(decimal.NewFromInt(int64(days)))
	fr.InterestAccrued = fr.InterestAccrued.Add(interest)
	fr.LastInterestCalc = time.Now()
	fr.UpdatedAt = time.Now()
}

func (fr *FinancingRecord) Repay(principal, interest decimal.Decimal) {
	fr.RepaidAmount = fr.RepaidAmount.Add(principal)
	fr.RepaidInterest = fr.RepaidInterest.Add(interest)
	fr.Outstanding = fr.Outstanding.Sub(principal)
	if fr.Outstanding.LessThanOrEqual(decimal.Zero) {
		fr.Outstanding = decimal.Zero
		now := time.Now()
		fr.Status = BorrowingStatusRepaid
		fr.RepaidAt = &now
	}
	fr.UpdatedAt = time.Now()
}

func (fr *FinancingRecord) Extend(newDueDate time.Time) {
	fr.DueDate = newDueDate
	fr.Status = BorrowingStatusExtended
	fr.UpdatedAt = time.Now()
}

func (fr *FinancingRecord) Default() {
	fr.Status = BorrowingStatusDefaulted
	fr.UpdatedAt = time.Now()
}

func (fr *FinancingRecord) IsOverdue() bool {
	return time.Now().After(fr.DueDate) && fr.Status == BorrowingStatusActive
}

type StockBorrow struct {
	ID               string          `json:"id"`
	AccountID        string          `json:"account_id"`
	Symbol           string          `json:"symbol"`
	Quantity         decimal.Decimal `json:"quantity"`
	BorrowPrice      decimal.Decimal `json:"borrow_price"`
	CurrentValue     decimal.Decimal `json:"current_value"`
	InterestRate     decimal.Decimal `json:"interest_rate"`
	InterestAccrued  decimal.Decimal `json:"interest_accrued"`
	RepaidQuantity   decimal.Decimal `json:"repaid_quantity"`
	OutstandingQty   decimal.Decimal `json:"outstanding_qty"`
	Status           BorrowingStatus `json:"status"`
	BorrowedAt       time.Time       `json:"borrowed_at"`
	DueDate          time.Time       `json:"due_date"`
	LastInterestCalc time.Time       `json:"last_interest_calc"`
	ReturnedAt       *time.Time      `json:"returned_at"`
	SourceAccount    string          `json:"source_account"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func NewStockBorrow(accountID, symbol string, quantity, borrowPrice, interestRate decimal.Decimal, termDays int, sourceAccount string) *StockBorrow {
	now := time.Now()
	return &StockBorrow{
		AccountID:        accountID,
		Symbol:           symbol,
		Quantity:         quantity,
		BorrowPrice:      borrowPrice,
		CurrentValue:     quantity.Mul(borrowPrice),
		InterestRate:     interestRate,
		InterestAccrued:  decimal.Zero,
		RepaidQuantity:   decimal.Zero,
		OutstandingQty:   quantity,
		Status:           BorrowingStatusActive,
		BorrowedAt:       now,
		DueDate:          now.AddDate(0, 0, termDays),
		LastInterestCalc: now,
		SourceAccount:    sourceAccount,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (sb *StockBorrow) UpdateCurrentValue(currentPrice decimal.Decimal) {
	sb.CurrentValue = sb.OutstandingQty.Mul(currentPrice)
	sb.UpdatedAt = time.Now()
}

func (sb *StockBorrow) AccrueInterest(days int) {
	dailyRate := sb.InterestRate.Div(decimal.NewFromInt(365))
	interest := sb.CurrentValue.Mul(dailyRate).Mul(decimal.NewFromInt(int64(days)))
	sb.InterestAccrued = sb.InterestAccrued.Add(interest)
	sb.LastInterestCalc = time.Now()
	sb.UpdatedAt = time.Now()
}

func (sb *StockBorrow) Return(quantity decimal.Decimal) {
	sb.RepaidQuantity = sb.RepaidQuantity.Add(quantity)
	sb.OutstandingQty = sb.OutstandingQty.Sub(quantity)
	if sb.OutstandingQty.LessThanOrEqual(decimal.Zero) {
		sb.OutstandingQty = decimal.Zero
		now := time.Now()
		sb.Status = BorrowingStatusRepaid
		sb.ReturnedAt = &now
	}
	sb.UpdatedAt = time.Now()
}

func (sb *StockBorrow) PayInterest(amount decimal.Decimal) {
	if sb.InterestAccrued.LessThan(amount) {
		amount = sb.InterestAccrued
	}
	sb.InterestAccrued = sb.InterestAccrued.Sub(amount)
	sb.UpdatedAt = time.Now()
}

func (sb *StockBorrow) Extend(newDueDate time.Time) {
	sb.DueDate = newDueDate
	sb.Status = BorrowingStatusExtended
	sb.UpdatedAt = time.Now()
}

func (sb *StockBorrow) IsOverdue() bool {
	return time.Now().After(sb.DueDate) && sb.Status == BorrowingStatusActive
}

type CollateralRecord struct {
	ID               string           `json:"id"`
	AccountID        string           `json:"account_id"`
	Symbol           string           `json:"symbol"`
	Quantity         decimal.Decimal  `json:"quantity"`
	LockPrice        decimal.Decimal  `json:"lock_price"`
	CurrentValue     decimal.Decimal  `json:"current_value"`
	Haircut          decimal.Decimal  `json:"haircut"`
	EligibleValue    decimal.Decimal  `json:"eligible_value"`
	Status           CollateralStatus `json:"status"`
	LockedAt         time.Time        `json:"locked_at"`
	ReleasedAt       *time.Time       `json:"released_at"`
	LastValuation    time.Time        `json:"last_valuation"`
	FinancingID      string           `json:"financing_id"`
	StockBorrowID    string           `json:"stock_borrow_id"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

func NewCollateralRecord(accountID, symbol string, quantity, price, haircut decimal.Decimal) *CollateralRecord {
	value := quantity.Mul(price)
	return &CollateralRecord{
		AccountID:     accountID,
		Symbol:        symbol,
		Quantity:      quantity,
		LockPrice:     price,
		CurrentValue:  value,
		Haircut:       haircut,
		EligibleValue: value.Mul(decimal.NewFromInt(1).Sub(haircut)),
		Status:        CollateralStatusLocked,
		LockedAt:      time.Now(),
		LastValuation: time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (cr *CollateralRecord) UpdateValue(currentPrice decimal.Decimal) {
	cr.CurrentValue = cr.Quantity.Mul(currentPrice)
	cr.EligibleValue = cr.CurrentValue.Mul(decimal.NewFromInt(1).Sub(cr.Haircut))
	cr.LastValuation = time.Now()
	cr.UpdatedAt = time.Now()
}

func (cr *CollateralRecord) Release() {
	now := time.Now()
	cr.Status = CollateralStatusReleased
	cr.ReleasedAt = &now
	cr.UpdatedAt = now
}

func (cr *CollateralRecord) Liquidate() {
	now := time.Now()
	cr.Status = CollateralStatusLiquidated
	cr.ReleasedAt = &now
	cr.UpdatedAt = now
}

type InterestRate struct {
	ID           string          `json:"id"`
	RateType     string          `json:"rate_type"`
	Symbol       string          `json:"symbol"`
	BaseRate     decimal.Decimal `json:"base_rate"`
	SpreadRate   decimal.Decimal `json:"spread_rate"`
	TotalRate    decimal.Decimal `json:"total_rate"`
	EffectiveFrom time.Time       `json:"effective_from"`
	EffectiveTo   *time.Time      `json:"effective_to"`
	IsActive      bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func NewInterestRate(rateType, symbol string, baseRate, spreadRate decimal.Decimal) *InterestRate {
	return &InterestRate{
		RateType:     rateType,
		Symbol:       symbol,
		BaseRate:     baseRate,
		SpreadRate:   spreadRate,
		TotalRate:    baseRate.Add(spreadRate),
		EffectiveFrom: time.Now(),
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (ir *InterestRate) UpdateRate(baseRate, spreadRate decimal.Decimal) {
	ir.BaseRate = baseRate
	ir.SpreadRate = spreadRate
	ir.TotalRate = baseRate.Add(spreadRate)
	ir.UpdatedAt = time.Now()
}

func (ir *InterestRate) Deactivate() {
	now := time.Now()
	ir.IsActive = false
	ir.EffectiveTo = &now
	ir.UpdatedAt = now
}

type MarginCall struct {
	ID               string           `json:"id"`
	AccountID        string           `json:"account_id"`
	MaintenanceRatio decimal.Decimal  `json:"maintenance_ratio"`
	RequiredRatio    decimal.Decimal  `json:"required_ratio"`
	DeficiencyAmount decimal.Decimal  `json:"deficiency_amount"`
	Status           string           `json:"status"`
	Deadline         time.Time        `json:"deadline"`
	NotifiedAt       *time.Time       `json:"notified_at"`
	ResolvedAt       *time.Time       `json:"resolved_at"`
	Actions          []MarginCallAction `json:"actions"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type MarginCallAction struct {
	ActionType  string          `json:"action_type"`
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
	TakenAt     time.Time       `json:"taken_at"`
}

func NewMarginCall(accountID string, maintenanceRatio, requiredRatio, deficiencyAmount decimal.Decimal, deadlineDuration time.Duration) *MarginCall {
	return &MarginCall{
		AccountID:        accountID,
		MaintenanceRatio: maintenanceRatio,
		RequiredRatio:    requiredRatio,
		DeficiencyAmount: deficiencyAmount,
		Status:           "PENDING",
		Deadline:         time.Now().Add(deadlineDuration),
		Actions:          []MarginCallAction{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (mc *MarginCall) Notify() {
	now := time.Now()
	mc.Status = "NOTIFIED"
	mc.NotifiedAt = &now
	mc.UpdatedAt = now
}

func (mc *MarginCall) Resolve() {
	now := time.Now()
	mc.Status = "RESOLVED"
	mc.ResolvedAt = &now
	mc.UpdatedAt = now
}

func (mc *MarginCall) IsExpired() bool {
	return time.Now().After(mc.Deadline)
}

func (mc *MarginCall) AddAction(actionType, description string, amount decimal.Decimal) {
	action := MarginCallAction{
		ActionType:  actionType,
		Amount:      amount,
		Description: description,
		TakenAt:     time.Now(),
	}
	mc.Actions = append(mc.Actions, action)
	mc.UpdatedAt = time.Now()
}

type StockInventory struct {
	ID             string          `json:"id"`
	Symbol         string          `json:"symbol"`
	TotalQuantity  decimal.Decimal `json:"total_quantity"`
	AvailableQty   decimal.Decimal `json:"available_qty"`
	BorrowedQty    decimal.Decimal `json:"borrowed_qty"`
	ReservedQty    decimal.Decimal `json:"reserved_qty"`
	BorrowRate     decimal.Decimal `json:"borrow_rate"`
	LastUpdated    time.Time       `json:"last_updated"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func NewStockInventory(symbol string, totalQuantity, borrowRate decimal.Decimal) *StockInventory {
	return &StockInventory{
		Symbol:        symbol,
		TotalQuantity: totalQuantity,
		AvailableQty:  totalQuantity,
		BorrowedQty:   decimal.Zero,
		ReservedQty:   decimal.Zero,
		BorrowRate:    borrowRate,
		LastUpdated:   time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (si *StockInventory) Borrow(quantity decimal.Decimal) error {
	if si.AvailableQty.LessThan(quantity) {
		return ErrStockUnavailable
	}
	si.AvailableQty = si.AvailableQty.Sub(quantity)
	si.BorrowedQty = si.BorrowedQty.Add(quantity)
	si.LastUpdated = time.Now()
	si.UpdatedAt = time.Now()
	return nil
}

func (si *StockInventory) Return(quantity decimal.Decimal) {
	si.BorrowedQty = si.BorrowedQty.Sub(quantity)
	si.AvailableQty = si.AvailableQty.Add(quantity)
	si.LastUpdated = time.Now()
	si.UpdatedAt = time.Now()
}

func (si *StockInventory) Reserve(quantity decimal.Decimal) error {
	if si.AvailableQty.LessThan(quantity) {
		return ErrStockUnavailable
	}
	si.AvailableQty = si.AvailableQty.Sub(quantity)
	si.ReservedQty = si.ReservedQty.Add(quantity)
	si.UpdatedAt = time.Now()
	return nil
}

func (si *StockInventory) ReleaseReservation(quantity decimal.Decimal) {
	si.ReservedQty = si.ReservedQty.Sub(quantity)
	si.AvailableQty = si.AvailableQty.Add(quantity)
	si.UpdatedAt = time.Now()
}

func (si *StockInventory) AddStock(quantity decimal.Decimal) {
	si.TotalQuantity = si.TotalQuantity.Add(quantity)
	si.AvailableQty = si.AvailableQty.Add(quantity)
	si.LastUpdated = time.Now()
	si.UpdatedAt = time.Now()
}

type FinancingApplication struct {
	ID              string          `json:"id"`
	AccountID       string          `json:"account_id"`
	FinancingType   FinancingType   `json:"financing_type"`
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	ApprovedAmount  decimal.Decimal `json:"approved_amount"`
	Purpose         string          `json:"purpose"`
	Status          string          `json:"status"`
	CollateralIDs   []string        `json:"collateral_ids"`
	RiskScore       decimal.Decimal `json:"risk_score"`
	AppliedAt       time.Time       `json:"applied_at"`
	ReviewedAt      *time.Time      `json:"reviewed_at"`
	ApprovedAt      *time.Time      `json:"approved_at"`
	RejectedAt      *time.Time      `json:"rejected_at"`
	RejectionReason string          `json:"rejection_reason"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func NewFinancingApplication(accountID string, financingType FinancingType, amount decimal.Decimal, purpose string) *FinancingApplication {
	return &FinancingApplication{
		AccountID:       accountID,
		FinancingType:   financingType,
		RequestedAmount: amount,
		ApprovedAmount:  decimal.Zero,
		Purpose:         purpose,
		Status:          "PENDING",
		CollateralIDs:   []string{},
		RiskScore:       decimal.Zero,
		AppliedAt:       time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func (fa *FinancingApplication) Approve(amount decimal.Decimal) {
	now := time.Now()
	fa.Status = "APPROVED"
	fa.ApprovedAmount = amount
	fa.ApprovedAt = &now
	fa.ReviewedAt = &now
	fa.UpdatedAt = now
}

func (fa *FinancingApplication) Reject(reason string) {
	now := time.Now()
	fa.Status = "REJECTED"
	fa.RejectionReason = reason
	fa.RejectedAt = &now
	fa.ReviewedAt = &now
	fa.UpdatedAt = now
}

func (fa *FinancingApplication) SetRiskScore(score decimal.Decimal) {
	fa.RiskScore = score
	fa.UpdatedAt = time.Now()
}

func (fa *FinancingApplication) AddCollateral(collateralID string) {
	fa.CollateralIDs = append(fa.CollateralIDs, collateralID)
	fa.UpdatedAt = time.Now()
}

type FinancingAccountRepository interface {
	Create(ctx context.Context, account *FinancingAccount) error
	Update(ctx context.Context, account *FinancingAccount) error
	FindByID(ctx context.Context, id string) (*FinancingAccount, error)
	FindByAccountID(ctx context.Context, accountID string) (*FinancingAccount, error)
	FindByStatus(ctx context.Context, status FinancingAccountStatus, limit, offset int) ([]*FinancingAccount, int64, error)
	FindRequiringMarginCall(ctx context.Context, threshold decimal.Decimal) ([]*FinancingAccount, error)
	Delete(ctx context.Context, id string) error
}

type FinancingRecordRepository interface {
	Create(ctx context.Context, record *FinancingRecord) error
	Update(ctx context.Context, record *FinancingRecord) error
	FindByID(ctx context.Context, id string) (*FinancingRecord, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*FinancingRecord, error)
	FindActive(ctx context.Context, accountID string) ([]*FinancingRecord, error)
	FindOverdue(ctx context.Context) ([]*FinancingRecord, error)
	Delete(ctx context.Context, id string) error
}

type StockBorrowRepository interface {
	Create(ctx context.Context, borrow *StockBorrow) error
	Update(ctx context.Context, borrow *StockBorrow) error
	FindByID(ctx context.Context, id string) (*StockBorrow, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*StockBorrow, error)
	FindBySymbol(ctx context.Context, accountID, symbol string) ([]*StockBorrow, error)
	FindActive(ctx context.Context, accountID string) ([]*StockBorrow, error)
	FindOverdue(ctx context.Context) ([]*StockBorrow, error)
	Delete(ctx context.Context, id string) error
}

type CollateralRecordRepository interface {
	Create(ctx context.Context, collateral *CollateralRecord) error
	Update(ctx context.Context, collateral *CollateralRecord) error
	FindByID(ctx context.Context, id string) (*CollateralRecord, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*CollateralRecord, error)
	FindByStatus(ctx context.Context, status CollateralStatus, limit, offset int) ([]*CollateralRecord, int64, error)
	Delete(ctx context.Context, id string) error
}

type InterestRateRepository interface {
	Create(ctx context.Context, rate *InterestRate) error
	Update(ctx context.Context, rate *InterestRate) error
	FindByID(ctx context.Context, id string) (*InterestRate, error)
	FindByTypeAndSymbol(ctx context.Context, rateType, symbol string) (*InterestRate, error)
	FindActive(ctx context.Context, rateType string) ([]*InterestRate, error)
	Delete(ctx context.Context, id string) error
}

type MarginCallRepository interface {
	Create(ctx context.Context, marginCall *MarginCall) error
	Update(ctx context.Context, marginCall *MarginCall) error
	FindByID(ctx context.Context, id string) (*MarginCall, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*MarginCall, error)
	FindPending(ctx context.Context, limit, offset int) ([]*MarginCall, int64, error)
	Delete(ctx context.Context, id string) error
}

type StockInventoryRepository interface {
	Create(ctx context.Context, inventory *StockInventory) error
	Update(ctx context.Context, inventory *StockInventory) error
	FindByID(ctx context.Context, id string) (*StockInventory, error)
	FindBySymbol(ctx context.Context, symbol string) (*StockInventory, error)
	FindAll(ctx context.Context, limit, offset int) ([]*StockInventory, int64, error)
	Delete(ctx context.Context, id string) error
}

type FinancingApplicationRepository interface {
	Create(ctx context.Context, application *FinancingApplication) error
	Update(ctx context.Context, application *FinancingApplication) error
	FindByID(ctx context.Context, id string) (*FinancingApplication, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*FinancingApplication, error)
	FindPending(ctx context.Context, limit, offset int) ([]*FinancingApplication, int64, error)
	Delete(ctx context.Context, id string) error
}
