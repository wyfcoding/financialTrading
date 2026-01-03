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
		Version:          0,
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

	if err := m.accountRepo.UpdateBalance(ctx, accountID, newBalance, newAvailable, account.FrozenBalance, account.Version); err != nil {
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

	return m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen, account.Version)
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

	return m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen, account.Version)
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

	if err := m.accountRepo.UpdateBalance(ctx, accountID, newBalance, account.AvailableBalance, newFrozen, account.Version); err != nil {
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

// --- TCC Distributed Transaction Support ---

// TccTryFreeze TCC Try: 预冻结资金
func (m *AccountManager) TccTryFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 1. 查找账户
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}

		// 2. 检查余额
		if account.AvailableBalance.LessThan(amount) {
			return fmt.Errorf("insufficient available balance")
		}

		// 3. 执行冻结 (Available -> Frozen)
		newAvailable := account.AvailableBalance.Sub(amount)
		newFrozen := account.FrozenBalance.Add(amount)

		return m.accountRepo.UpdateBalance(ctx, account.AccountID, account.Balance, newAvailable, newFrozen, account.Version)
	})
}

// TccConfirmFreeze TCC Confirm: 确认冻结
func (m *AccountManager) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return nil
	})
}

// TccCancelFreeze TCC Cancel: 取消冻结 (Frozen -> Available)
func (m *AccountManager) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 1. 查找账户
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}

		// 2. 执行解冻 (Frozen -> Available)
		if account.FrozenBalance.LessThan(amount) {
			return fmt.Errorf("frozen balance mismatch during cancel")
		}

		newAvailable := account.AvailableBalance.Add(amount)
		newFrozen := account.FrozenBalance.Sub(amount)

		return m.accountRepo.UpdateBalance(ctx, account.AccountID, account.Balance, newAvailable, newFrozen, account.Version)
	})
}

// --- Saga Distributed Transaction Support ---

// SagaDeductFrozen Saga 正向: 扣除冻结资金 (成交确认，资产永久转出)
func (m *AccountManager) SagaDeductFrozen(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		if account.FrozenBalance.LessThan(amount) {
			return fmt.Errorf("insufficient frozen balance for deduction: user=%s, currency=%s", userID, currency)
		}

		newBalance := account.Balance.Sub(amount)
		newFrozen := account.FrozenBalance.Sub(amount)

		if err := m.accountRepo.UpdateBalance(ctx, account.AccountID, newBalance, account.AvailableBalance, newFrozen, account.Version); err != nil {
			return err
		}

		return m.transactionRepo.Save(ctx, &domain.Transaction{
			TransactionID: fmt.Sprintf("SAGA-DED-%d", idgen.GenID()),
			AccountID:     account.AccountID,
			Type:          "SETTLE_OUT",
			Amount:        amount.Neg(),
			Status:        "COMPLETED",
		})
	})
}

// SagaRefundFrozen Saga 补偿: 恢复冻结资金 (回滚扣除操作)
func (m *AccountManager) SagaRefundFrozen(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		newBalance := account.Balance.Add(amount)
		newFrozen := account.FrozenBalance.Add(amount)
		return m.accountRepo.UpdateBalance(ctx, account.AccountID, newBalance, account.AvailableBalance, newFrozen, account.Version)
	})
}

// SagaAddBalance Saga 正向: 增加余额 (成交确认，资产入账)
func (m *AccountManager) SagaAddBalance(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		newBalance := account.Balance.Add(amount)
		newAvailable := account.AvailableBalance.Add(amount)

		if err := m.accountRepo.UpdateBalance(ctx, account.AccountID, newBalance, newAvailable, account.FrozenBalance, account.Version); err != nil {
			return err
		}

		return m.transactionRepo.Save(ctx, &domain.Transaction{
			TransactionID: fmt.Sprintf("SAGA-ADD-%d", idgen.GenID()),
			AccountID:     account.AccountID,
			Type:          "SETTLE_IN",
			Amount:        amount,
			Status:        "COMPLETED",
		})
	})
}

// SagaSubBalance Saga 补偿: 扣除已增加的余额 (回滚入账操作)
func (m *AccountManager) SagaSubBalance(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		if account.AvailableBalance.LessThan(amount) {
			return fmt.Errorf("insufficient available balance for compensation: user=%s, currency=%s", userID, currency)
		}

		newBalance := account.Balance.Sub(amount)
		newAvailable := account.AvailableBalance.Sub(amount)
		return m.accountRepo.UpdateBalance(ctx, account.AccountID, newBalance, newAvailable, account.FrozenBalance, account.Version)
	})
}

func (m *AccountManager) getAccount(ctx context.Context, userID, currency string) (*domain.Account, error) {
	accounts, err := m.accountRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, acc := range accounts {
		if acc.Currency == currency {
			return acc, nil
		}
	}
	return nil, fmt.Errorf("account not found: user=%s, currency=%s", userID, currency)
}
