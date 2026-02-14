package domain

import (
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/eventsourcing"
)

var (
	ErrCurrencyNotSupported = errors.New("currency not supported")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrAccountNotFound      = errors.New("account not found")
	ErrInvalidExchangeRate  = errors.New("invalid exchange rate")
	ErrCrossCurrencyNotAllowed = errors.New("cross currency operation not allowed")
)

type CurrencyType string

const (
	CurrencyTypeFiat     CurrencyType = "FIAT"
	CurrencyTypeCrypto   CurrencyType = "CRYPTO"
	CurrencyTypeStablecoin CurrencyType = "STABLECOIN"
)

type CurrencyConfig struct {
	Code         string       `json:"code"`
	Name         string       `json:"name"`
	Symbol       string       `json:"symbol"`
	Type         CurrencyType `json:"type"`
	Decimals     int          `json:"decimals"`
	MinAmount    decimal.Decimal `json:"min_amount"`
	MaxAmount    decimal.Decimal `json:"max_amount"`
	IsBase       bool         `json:"is_base"`
	IsEnabled    bool         `json:"is_enabled"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
}

type MultiCurrencyAccount struct {
	eventsourcing.AggregateRoot
	ID               uint                     `json:"id"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	AccountID        string                   `json:"account_id"`
	UserID           string                   `json:"user_id"`
	AccountType      AccountType              `json:"account_type"`
	BaseCurrency     string                   `json:"base_currency"`
	CurrencyAccounts map[string]*CurrencyAccount `json:"currency_accounts"`
	VIPLevel         int                      `json:"vip_level"`
	Status           string                   `json:"status"`
}

type CurrencyAccount struct {
	Currency         string          `json:"currency"`
	Balance          decimal.Decimal `json:"balance"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	FrozenBalance    decimal.Decimal `json:"frozen_balance"`
	BorrowedAmount   decimal.Decimal `json:"borrowed_amount"`
	LockedCollateral decimal.Decimal `json:"locked_collateral"`
	AccruedInterest  decimal.Decimal `json:"accrued_interest"`
	LastUpdated      time.Time       `json:"last_updated"`
}

type ExchangeRate struct {
	BaseCurrency   string          `json:"base_currency"`
	QuoteCurrency  string          `json:"quote_currency"`
	Rate           decimal.Decimal `json:"rate"`
	BidRate        decimal.Decimal `json:"bid_rate"`
	AskRate        decimal.Decimal `json:"ask_rate"`
	Source         string          `json:"source"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ValidUntil     time.Time       `json:"valid_until"`
}

type ExchangeRateProvider interface {
	GetRate(baseCurrency, quoteCurrency string) (*ExchangeRate, error)
	GetAllRates(baseCurrency string) (map[string]*ExchangeRate, error)
	RefreshRates() error
}

type CurrencyConversion struct {
	ID              string          `json:"id"`
	AccountID       string          `json:"account_id"`
	UserID          string          `json:"user_id"`
	FromCurrency    string          `json:"from_currency"`
	ToCurrency      string          `json:"to_currency"`
	FromAmount      decimal.Decimal `json:"from_amount"`
	ToAmount        decimal.Decimal `json:"to_amount"`
	ExchangeRate    decimal.Decimal `json:"exchange_rate"`
	Fee             decimal.Decimal `json:"fee"`
	FeeCurrency     string          `json:"fee_currency"`
	Status          string          `json:"status"`
	ConvertedAt     time.Time       `json:"converted_at"`
}

type CrossCurrencyTransfer struct {
	ID             string          `json:"id"`
	FromAccountID  string          `json:"from_account_id"`
	ToAccountID    string          `json:"to_account_id"`
	FromCurrency   string          `json:"from_currency"`
	ToCurrency     string          `json:"to_currency"`
	FromAmount     decimal.Decimal `json:"from_amount"`
	ToAmount       decimal.Decimal `json:"to_amount"`
	ExchangeRate   decimal.Decimal `json:"exchange_rate"`
	Status         string          `json:"status"`
	TransferredAt  time.Time       `json:"transferred_at"`
}

func NewMultiCurrencyAccount(accountID, userID, baseCurrency string, accType AccountType) *MultiCurrencyAccount {
	now := time.Now()
	return &MultiCurrencyAccount{
		AccountID:        accountID,
		UserID:           userID,
		AccountType:      accType,
		BaseCurrency:     baseCurrency,
		CurrencyAccounts: make(map[string]*CurrencyAccount),
		Status:           "ACTIVE",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (a *MultiCurrencyAccount) GetOrCreateCurrencyAccount(currency string) *CurrencyAccount {
	if acc, exists := a.CurrencyAccounts[currency]; exists {
		return acc
	}
	
	acc := &CurrencyAccount{
		Currency:         currency,
		Balance:          decimal.Zero,
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
		BorrowedAmount:   decimal.Zero,
		LockedCollateral: decimal.Zero,
		AccruedInterest:  decimal.Zero,
		LastUpdated:      time.Now(),
	}
	a.CurrencyAccounts[currency] = acc
	return acc
}

func (a *MultiCurrencyAccount) GetCurrencyAccount(currency string) (*CurrencyAccount, error) {
	acc, exists := a.CurrencyAccounts[currency]
	if !exists {
		return nil, ErrAccountNotFound
	}
	return acc, nil
}

func (a *MultiCurrencyAccount) Deposit(currency string, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount must be positive")
	}
	
	acc := a.GetOrCreateCurrencyAccount(currency)
	acc.Balance = acc.Balance.Add(amount)
	acc.AvailableBalance = acc.AvailableBalance.Add(amount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) Withdraw(currency string, amount decimal.Decimal) error {
	acc, err := a.GetCurrencyAccount(currency)
	if err != nil {
		return err
	}
	
	if acc.AvailableBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}
	
	acc.Balance = acc.Balance.Sub(amount)
	acc.AvailableBalance = acc.AvailableBalance.Sub(amount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) Freeze(currency string, amount decimal.Decimal, reason string) error {
	acc, err := a.GetCurrencyAccount(currency)
	if err != nil {
		return err
	}
	
	if acc.AvailableBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}
	
	acc.AvailableBalance = acc.AvailableBalance.Sub(amount)
	acc.FrozenBalance = acc.FrozenBalance.Add(amount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) Unfreeze(currency string, amount decimal.Decimal) error {
	acc, err := a.GetCurrencyAccount(currency)
	if err != nil {
		return err
	}
	
	if acc.FrozenBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}
	
	acc.FrozenBalance = acc.FrozenBalance.Sub(amount)
	acc.AvailableBalance = acc.AvailableBalance.Add(amount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) DeductFrozen(currency string, amount decimal.Decimal) error {
	acc, err := a.GetCurrencyAccount(currency)
	if err != nil {
		return err
	}
	
	if acc.FrozenBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}
	
	acc.Balance = acc.Balance.Sub(amount)
	acc.FrozenBalance = acc.FrozenBalance.Sub(amount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) ConvertCurrency(
	fromCurrency, toCurrency string,
	fromAmount decimal.Decimal,
	rateProvider ExchangeRateProvider,
) (*CurrencyConversion, error) {
	if fromCurrency == toCurrency {
		return nil, errors.New("same currency conversion not needed")
	}
	
	fromAcc, err := a.GetCurrencyAccount(fromCurrency)
	if err != nil {
		return nil, err
	}
	
	if fromAcc.AvailableBalance.LessThan(fromAmount) {
		return nil, ErrInsufficientBalance
	}
	
	rate, err := rateProvider.GetRate(fromCurrency, toCurrency)
	if err != nil {
		return nil, err
	}
	
	toAmount := fromAmount.Mul(rate.Rate)
	fee := toAmount.Mul(decimal.NewFromFloat(0.001))
	toAmount = toAmount.Sub(fee)
	
	fromAcc.AvailableBalance = fromAcc.AvailableBalance.Sub(fromAmount)
	fromAcc.Balance = fromAcc.Balance.Sub(fromAmount)
	fromAcc.LastUpdated = time.Now()
	
	toAcc := a.GetOrCreateCurrencyAccount(toCurrency)
	toAcc.AvailableBalance = toAcc.AvailableBalance.Add(toAmount)
	toAcc.Balance = toAcc.Balance.Add(toAmount)
	toAcc.LastUpdated = time.Now()
	
	a.UpdatedAt = time.Now()
	
	conversion := &CurrencyConversion{
		ID:           generateConversionID(),
		AccountID:    a.AccountID,
		UserID:       a.UserID,
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		FromAmount:   fromAmount,
		ToAmount:     toAmount,
		ExchangeRate: rate.Rate,
		Fee:          fee,
		FeeCurrency:  toCurrency,
		Status:       "COMPLETED",
		ConvertedAt:  time.Now(),
	}
	
	return conversion, nil
}

func (a *MultiCurrencyAccount) GetTotalBalanceInBaseCurrency(rateProvider ExchangeRateProvider) (decimal.Decimal, error) {
	total := decimal.Zero
	
	for currency, acc := range a.CurrencyAccounts {
		if currency == a.BaseCurrency {
			total = total.Add(acc.Balance)
		} else {
			rate, err := rateProvider.GetRate(currency, a.BaseCurrency)
			if err != nil {
				continue
			}
			total = total.Add(acc.Balance.Mul(rate.Rate))
		}
	}
	
	return total, nil
}

func (a *MultiCurrencyAccount) GetAvailableBalanceInBaseCurrency(rateProvider ExchangeRateProvider) (decimal.Decimal, error) {
	total := decimal.Zero
	
	for currency, acc := range a.CurrencyAccounts {
		if currency == a.BaseCurrency {
			total = total.Add(acc.AvailableBalance)
		} else {
			rate, err := rateProvider.GetRate(currency, a.BaseCurrency)
			if err != nil {
				continue
			}
			total = total.Add(acc.AvailableBalance.Mul(rate.Rate))
		}
	}
	
	return total, nil
}

func (a *MultiCurrencyAccount) TransferTo(
	toAccount *MultiCurrencyAccount,
	fromCurrency, toCurrency string,
	amount decimal.Decimal,
	rateProvider ExchangeRateProvider,
) (*CrossCurrencyTransfer, error) {
	if err := a.Withdraw(fromCurrency, amount); err != nil {
		return nil, err
	}
	
	var toAmount decimal.Decimal
	if fromCurrency == toCurrency {
		toAmount = amount
	} else {
		rate, err := rateProvider.GetRate(fromCurrency, toCurrency)
		if err != nil {
			a.Deposit(fromCurrency, amount)
			return nil, err
		}
		toAmount = amount.Mul(rate.Rate)
	}
	
	if err := toAccount.Deposit(toCurrency, toAmount); err != nil {
		a.Deposit(fromCurrency, amount)
		return nil, err
	}
	
	transfer := &CrossCurrencyTransfer{
		ID:            generateTransferID(),
		FromAccountID: a.AccountID,
		ToAccountID:   toAccount.AccountID,
		FromCurrency:  fromCurrency,
		ToCurrency:    toCurrency,
		FromAmount:    amount,
		ToAmount:      toAmount,
		Status:        "COMPLETED",
		TransferredAt: time.Now(),
	}
	
	return transfer, nil
}

func (a *MultiCurrencyAccount) Borrow(currency string, amount decimal.Decimal) error {
	if a.AccountType != AccountTypeMargin {
		return errors.New("only margin accounts can borrow")
	}
	
	acc := a.GetOrCreateCurrencyAccount(currency)
	acc.Balance = acc.Balance.Add(amount)
	acc.AvailableBalance = acc.AvailableBalance.Add(amount)
	acc.BorrowedAmount = acc.BorrowedAmount.Add(amount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) Repay(currency string, amount decimal.Decimal) error {
	acc, err := a.GetCurrencyAccount(currency)
	if err != nil {
		return err
	}
	
	if acc.AvailableBalance.LessThan(amount) {
		return ErrInsufficientBalance
	}
	
	repayAmount := amount
	if repayAmount.GreaterThan(acc.BorrowedAmount) {
		repayAmount = acc.BorrowedAmount
	}
	
	acc.AvailableBalance = acc.AvailableBalance.Sub(amount)
	acc.Balance = acc.Balance.Sub(amount)
	acc.BorrowedAmount = acc.BorrowedAmount.Sub(repayAmount)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
	
	return nil
}

func (a *MultiCurrencyAccount) AccrueInterest(currency string, rate decimal.Decimal) {
	acc, exists := a.CurrencyAccounts[currency]
	if !exists || !acc.BorrowedAmount.IsPositive() {
		return
	}
	
	interest := acc.BorrowedAmount.Mul(rate)
	acc.AccruedInterest = acc.AccruedInterest.Add(interest)
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
}

func (a *MultiCurrencyAccount) SettleInterest(currency string) {
	acc, exists := a.CurrencyAccounts[currency]
	if !exists || acc.AccruedInterest.IsZero() {
		return
	}
	
	if a.AccountType == AccountTypeMargin {
		acc.BorrowedAmount = acc.BorrowedAmount.Add(acc.AccruedInterest)
	}
	
	acc.AccruedInterest = decimal.Zero
	acc.LastUpdated = time.Now()
	a.UpdatedAt = time.Now()
}

type MultiCurrencyAccountService struct {
	rateProvider ExchangeRateProvider
	accountRepo  MultiCurrencyAccountRepository
	mu           sync.RWMutex
}

func NewMultiCurrencyAccountService(
	rateProvider ExchangeRateProvider,
	accountRepo MultiCurrencyAccountRepository,
) *MultiCurrencyAccountService {
	return &MultiCurrencyAccountService{
		rateProvider: rateProvider,
		accountRepo:  accountRepo,
	}
}

func (s *MultiCurrencyAccountService) CreateAccount(
	ctx interface{},
	userID, baseCurrency string,
	accountType AccountType,
) (*MultiCurrencyAccount, error) {
	accountID := generateAccountID()
	account := NewMultiCurrencyAccount(accountID, userID, baseCurrency, accountType)
	account.GetOrCreateCurrencyAccount(baseCurrency)
	
	if err := s.accountRepo.Save(account); err != nil {
		return nil, err
	}
	
	return account, nil
}

func (s *MultiCurrencyAccountService) Deposit(
	ctx interface{},
	accountID, currency string,
	amount decimal.Decimal,
) error {
	account, err := s.accountRepo.FindByID(accountID)
	if err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if err := account.Deposit(currency, amount); err != nil {
		return err
	}
	
	return s.accountRepo.Save(account)
}

func (s *MultiCurrencyAccountService) ConvertCurrency(
	ctx interface{},
	accountID, fromCurrency, toCurrency string,
	amount decimal.Decimal,
) (*CurrencyConversion, error) {
	account, err := s.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	conversion, err := account.ConvertCurrency(fromCurrency, toCurrency, amount, s.rateProvider)
	if err != nil {
		return nil, err
	}
	
	if err := s.accountRepo.Save(account); err != nil {
		return nil, err
	}
	
	return conversion, nil
}

func (s *MultiCurrencyAccountService) GetTotalBalance(
	ctx interface{},
	accountID string,
) (map[string]decimal.Decimal, error) {
	account, err := s.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, err
	}
	
	balances := make(map[string]decimal.Decimal)
	for currency, acc := range account.CurrencyAccounts {
		balances[currency] = acc.Balance
	}
	
	return balances, nil
}

func (s *MultiCurrencyAccountService) GetTotalBalanceInBase(
	ctx interface{},
	accountID string,
) (decimal.Decimal, error) {
	account, err := s.accountRepo.FindByID(accountID)
	if err != nil {
		return decimal.Zero, err
	}
	
	return account.GetTotalBalanceInBaseCurrency(s.rateProvider)
}

type MultiCurrencyAccountRepository interface {
	Save(account *MultiCurrencyAccount) error
	FindByID(accountID string) (*MultiCurrencyAccount, error)
	FindByUserID(userID string) ([]*MultiCurrencyAccount, error)
	Update(account *MultiCurrencyAccount) error
	Delete(accountID string) error
}

type CurrencyConversionRepository interface {
	Save(conversion *CurrencyConversion) error
	FindByID(id string) (*CurrencyConversion, error)
	FindByAccountID(accountID string, startTime, endTime *time.Time) ([]*CurrencyConversion, error)
}

type ExchangeRateRepository interface {
	Save(rate *ExchangeRate) error
	Find(baseCurrency, quoteCurrency string) (*ExchangeRate, error)
	FindAllByBase(baseCurrency string) ([]*ExchangeRate, error)
}

func generateConversionID() string {
	return "CONV" + time.Now().Format("20060102150405")
}

func generateTransferID() string {
	return "XFERT" + time.Now().Format("20060102150405")
}

func generateAccountID() string {
	return "ACC" + time.Now().Format("20060102150405")
}
