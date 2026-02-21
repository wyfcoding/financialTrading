package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
	"github.com/wyfcoding/pkg/idgen"
	"log/slog"
)

type LedgerCommandService struct {
	repo   domain.LedgerRepository
	logger *slog.Logger
	mu     sync.Mutex
}

func NewLedgerCommandService(repo domain.LedgerRepository, logger *slog.Logger) *LedgerCommandService {
	return &LedgerCommandService{repo: repo, logger: logger}
}

type TransferCommand struct {
	TransactionID string
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
	Currency      string
	Description   string
	PostedBy      string
}

func (s *LedgerCommandService) Transfer(ctx context.Context, cmd *TransferCommand) (*domain.Journal, error) {
	if cmd == nil {
		return nil, errors.New("transfer command is nil")
	}
	if cmd.FromAccountID == "" || cmd.ToAccountID == "" {
		return nil, errors.New("from_account_id and to_account_id are required")
	}
	if !cmd.Amount.IsPositive() {
		return nil, errors.New("amount must be positive")
	}
	currency := normalizeCurrency(cmd.Currency)
	txID := cmd.TransactionID
	if txID == "" {
		txID = idgen.GenIDString()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fromAccount, err := s.loadOrCreateAccount(ctx, cmd.FromAccountID, currency)
	if err != nil {
		return nil, err
	}
	toAccount, err := s.loadOrCreateAccount(ctx, cmd.ToAccountID, currency)
	if err != nil {
		return nil, err
	}
	if !fromAccount.IsActive() || !toAccount.IsActive() {
		return nil, domain.ErrAccountFrozen
	}
	if fromAccount.Currency != currency || toAccount.Currency != currency {
		return nil, domain.ErrCurrencyMismatch
	}
	if fromAccount.AvailableBalance().LessThan(cmd.Amount) {
		return nil, domain.ErrInsufficientBalance
	}

	fromAccount.Balance = fromAccount.Balance.Sub(cmd.Amount)
	fromAccount.UpdatedAt = time.Now()
	toAccount.Balance = toAccount.Balance.Add(cmd.Amount)
	toAccount.UpdatedAt = time.Now()

	now := time.Now()
	journal := &domain.Journal{
		ID:            idgen.GenIDString(),
		TransactionID: txID,
		JournalType:   domain.JournalTrade,
		PostedAt:      now,
		Entries: []domain.LedgerEntry{
			{
				ID:            idgen.GenIDString(),
				TransactionID: txID,
				AccountID:     cmd.FromAccountID,
				Direction:     domain.Credit,
				Currency:      currency,
				Amount:        cmd.Amount,
				BalanceAfter:  fromAccount.Balance,
				Narration:     cmd.Description,
			},
			{
				ID:            idgen.GenIDString(),
				TransactionID: txID,
				AccountID:     cmd.ToAccountID,
				Direction:     domain.Debit,
				Currency:      currency,
				Amount:        cmd.Amount,
				BalanceAfter:  toAccount.Balance,
				Narration:     cmd.Description,
			},
		},
		Context: map[string]any{
			"posted_by": cmd.PostedBy,
		},
	}
	if err := journal.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateBalance(ctx, fromAccount); err != nil {
		return nil, fmt.Errorf("update from account balance: %w", err)
	}
	if err := s.repo.UpdateBalance(ctx, toAccount); err != nil {
		return nil, fmt.Errorf("update to account balance: %w", err)
	}
	if err := s.repo.SaveJournal(ctx, journal); err != nil {
		return nil, err
	}
	return journal, nil
}

type HoldFundsCommand struct {
	AccountID   string
	ReferenceID string
	Amount      decimal.Decimal
	Currency    string
	Reason      string
	ExpiresAt   *time.Time
}

func (s *LedgerCommandService) HoldFunds(ctx context.Context, cmd *HoldFundsCommand) (*domain.FundHold, error) {
	if cmd == nil {
		return nil, errors.New("hold funds command is nil")
	}
	if cmd.AccountID == "" {
		return nil, errors.New("account_id is required")
	}
	if !cmd.Amount.IsPositive() {
		return nil, errors.New("amount must be positive")
	}
	currency := normalizeCurrency(cmd.Currency)
	reason := cmd.Reason
	if reason == "" {
		reason = "manual_hold"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	account, err := s.loadOrCreateAccount(ctx, cmd.AccountID, currency)
	if err != nil {
		return nil, err
	}
	if !account.IsActive() {
		return nil, domain.ErrAccountFrozen
	}
	if account.Currency != currency {
		return nil, domain.ErrCurrencyMismatch
	}
	if account.AvailableBalance().LessThan(cmd.Amount) {
		return nil, domain.ErrInsufficientBalance
	}

	now := time.Now()
	account.HoldBalance = account.HoldBalance.Add(cmd.Amount)
	account.UpdatedAt = now

	holdID := cmd.ReferenceID
	if holdID == "" {
		holdID = idgen.GenIDString()
	}
	expiresAt := now.Add(24 * time.Hour)
	if cmd.ExpiresAt != nil {
		expiresAt = *cmd.ExpiresAt
	}

	hold := &domain.FundHold{
		HoldID:    holdID,
		AccountID: cmd.AccountID,
		Amount:    cmd.Amount,
		Currency:  currency,
		Reason:    reason,
		Status:    "ACTIVE",
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.UpdateBalance(ctx, account); err != nil {
		return nil, err
	}
	if err := s.repo.CreateHold(ctx, hold); err != nil {
		return nil, err
	}
	return hold, nil
}

type ReleaseFundsCommand struct {
	ReferenceID string
	Reason      string
}

func (s *LedgerCommandService) ReleaseFunds(ctx context.Context, cmd *ReleaseFundsCommand) error {
	if cmd == nil || cmd.ReferenceID == "" {
		return errors.New("reference_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hold, err := s.repo.GetHoldForUpdate(ctx, cmd.ReferenceID)
	if err != nil {
		return err
	}
	if hold.Status != "ACTIVE" {
		return domain.ErrHoldAlreadyProcessed
	}

	account, err := s.repo.GetAccountForUpdate(ctx, hold.AccountID)
	if err != nil {
		return err
	}
	if account.HoldBalance.LessThan(hold.Amount) {
		return domain.ErrInsufficientBalance
	}
	account.HoldBalance = account.HoldBalance.Sub(hold.Amount)
	account.UpdatedAt = time.Now()
	if err := s.repo.UpdateBalance(ctx, account); err != nil {
		return err
	}

	hold.Status = "RELEASED"
	hold.UpdatedAt = time.Now()
	return s.repo.UpdateHold(ctx, hold)
}

func (s *LedgerCommandService) loadOrCreateAccount(ctx context.Context, accountID, currency string) (*domain.Account, error) {
	account, err := s.repo.GetAccountForUpdate(ctx, accountID)
	if err == nil {
		if account.Status == "" {
			account.Status = "ACTIVE"
		}
		if account.Currency == "" {
			account.Currency = currency
		}
		return account, nil
	}
	if !errors.Is(err, domain.ErrAccountNotFound) {
		return nil, err
	}

	now := time.Now()
	account = &domain.Account{
		ID:          accountID,
		Currency:    currency,
		Status:      "ACTIVE",
		Balance:     decimal.Zero,
		HoldBalance: decimal.Zero,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if saveErr := s.repo.UpdateBalance(ctx, account); saveErr != nil {
		return nil, saveErr
	}
	return account, nil
}

func normalizeCurrency(currency string) string {
	cur := strings.TrimSpace(currency)
	if cur == "" {
		return "USD"
	}
	return strings.ToUpper(cur)
}
