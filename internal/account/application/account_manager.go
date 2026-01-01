package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// AccountManager 处理所有账户相关的写入操作（Commands）。
type AccountManager struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
}

// NewAccountManager 构造函数。
func NewAccountManager(accountRepo domain.AccountRepository, transactionRepo domain.TransactionRepository) *AccountManager {
	return &AccountManager{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// CreateAccount 创建账户
func (m *AccountManager) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountDTO, error) {
	if req.UserID == "" || req.AccountType == "" || req.Currency == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	accountID := fmt.Sprintf("ACC-%d", idgen.GenID())

	account := &domain.Account{
		AccountID:        accountID,
		UserID:           req.UserID,
		AccountType:      req.AccountType,
		Currency:         req.Currency,
		Balance:          decimal.Zero,
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
	}

	if err := m.accountRepo.Save(ctx, account); err != nil {
		return nil, err
	}

	return &AccountDTO{
		AccountID:        account.AccountID,
		UserID:           account.UserID,
		AccountType:      account.AccountType,
		Currency:         account.Currency,
		Balance:          account.Balance.String(),
		AvailableBalance: account.AvailableBalance.String(),
		FrozenBalance:    account.FrozenBalance.String(),
		CreatedAt:        account.CreatedAt.Unix(),
		UpdatedAt:        account.UpdatedAt.Unix(),
	}, nil
}

// Deposit 充值
func (m *AccountManager) Deposit(ctx context.Context, accountID string, amount decimal.Decimal) error {
	account, err := m.accountRepo.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	newBalance := account.Balance.Add(amount)
	newAvailable := account.AvailableBalance.Add(amount)

	if err := m.accountRepo.UpdateBalance(ctx, accountID, newBalance, newAvailable, account.FrozenBalance); err != nil {
		return err
	}

	transaction := &domain.Transaction{
		TransactionID: fmt.Sprintf("TXN-%d", idgen.GenID()),
		AccountID:     accountID,
		Type:          "DEPOSIT",
		Amount:        amount,
		Status:        "COMPLETED",
	}

	return m.transactionRepo.Save(ctx, transaction)
}

// FreezeBalance 冻结余额
func (m *AccountManager) FreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal, reason string) error {
	account, err := m.accountRepo.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	if account.AvailableBalance.LessThan(amount) {
		return fmt.Errorf("insufficient available balance")
	}

	newAvailable := account.AvailableBalance.Sub(amount)
	newFrozen := account.FrozenBalance.Add(amount)

	return m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen)
}

// UnfreezeBalance 解冻余额
func (m *AccountManager) UnfreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	account, err := m.accountRepo.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	if account.FrozenBalance.LessThan(amount) {
		return fmt.Errorf("frozen balance insufficient to unfreeze")
	}

	newAvailable := account.AvailableBalance.Add(amount)
	newFrozen := account.FrozenBalance.Sub(amount)

	return m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen)
}

// DeductFrozenBalance 扣除已冻结的余额
func (m *AccountManager) DeductFrozenBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	account, err := m.accountRepo.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	if account.FrozenBalance.LessThan(amount) {
		return fmt.Errorf("frozen balance insufficient to deduct")
	}

	newBalance := account.Balance.Sub(amount)
	newFrozen := account.FrozenBalance.Sub(amount)

	if err := m.accountRepo.UpdateBalance(ctx, accountID, newBalance, account.AvailableBalance, newFrozen); err != nil {
		return err
	}

	transaction := &domain.Transaction{
		TransactionID: fmt.Sprintf("DED-%d", idgen.GenID()),
		AccountID:     accountID,
		Type:          "DEDUCT",
		Amount:        amount,
		Status:        "COMPLETED",
	}
	return m.transactionRepo.Save(ctx, transaction)
}
