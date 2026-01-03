package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

// AccountManager 处理所有账户相关的写入操作（Commands）。
type AccountManager struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
	outbox          *outbox.Manager
	db              *gorm.DB
}

// NewAccountManager 构造函数。
func NewAccountManager(
	accountRepo domain.AccountRepository,
	transactionRepo domain.TransactionRepository,
	outboxMgr *outbox.Manager,
	db *gorm.DB,
) *AccountManager {
	return &AccountManager{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		outbox:          outboxMgr,
		db:              db,
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
	return m.db.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx_db", tx)
		account, err := m.accountRepo.Get(txCtx, accountID)
		if err != nil || account == nil {
			return fmt.Errorf("account not found")
		}

		newBalance := account.Balance.Add(amount)
		newAvailable := account.AvailableBalance.Add(amount)

		if err := m.accountRepo.UpdateBalance(txCtx, accountID, newBalance, newAvailable, account.FrozenBalance, account.Version); err != nil {
			return err
		}

		transactionID := fmt.Sprintf("TXN-%d", idgen.GenID())
		if err := m.transactionRepo.Save(txCtx, &domain.Transaction{
			TransactionID: transactionID,
			AccountID:     accountID,
			Type:          "DEPOSIT",
			Amount:        amount,
			Status:        "COMPLETED",
		}); err != nil {
			return err
		}

		return m.outbox.PublishInTx(ctx, tx, "account.balance.changed", transactionID, map[string]any{
			"user_id": account.UserID, "currency": account.Currency, "change": amount.String(), "type": "DEPOSIT",
		})
	})
}

// FreezeBalance 冻结余额
func (m *AccountManager) FreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal, reason string) error {
	account, err := m.accountRepo.Get(ctx, accountID)
	if err != nil || account == nil {
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
	if err != nil || account == nil {
		return fmt.Errorf("account not found")
	}

	if account.FrozenBalance.LessThan(amount) {
		return fmt.Errorf("frozen balance insufficient")
	}

	newAvailable := account.AvailableBalance.Add(amount)
	newFrozen := account.FrozenBalance.Sub(amount)

	return m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen, account.Version)
}

// DeductFrozenBalance 扣除已冻结的余额
func (m *AccountManager) DeductFrozenBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx_db", tx)
		account, err := m.accountRepo.Get(txCtx, accountID)
		if err != nil || account == nil {
			return fmt.Errorf("account not found")
		}
		if account.FrozenBalance.LessThan(amount) {
			return fmt.Errorf("frozen balance insufficient to deduct")
		}
		newBalance := account.Balance.Sub(amount)
		newFrozen := account.FrozenBalance.Sub(amount)

		if err := m.accountRepo.UpdateBalance(txCtx, accountID, newBalance, account.AvailableBalance, newFrozen, account.Version); err != nil {
			return err
		}

		transactionID := fmt.Sprintf("DED-%d", idgen.GenID())
		return m.transactionRepo.Save(txCtx, &domain.Transaction{
			TransactionID: transactionID, AccountID: accountID, Type: "DEDUCT", Amount: amount, Status: "COMPLETED",
		})
	})
}

// --- TCC Distributed Transaction Support ---

func (m *AccountManager) TccTryFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		if account.AvailableBalance.LessThan(amount) {
			return fmt.Errorf("insufficient available balance")
		}

		newAvailable := account.AvailableBalance.Sub(amount)
		newFrozen := account.FrozenBalance.Add(amount)
		return m.accountRepo.UpdateBalance(ctx, account.AccountID, account.Balance, newAvailable, newFrozen, account.Version)
	})
}

func (m *AccountManager) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error { return nil })
}

func (m *AccountManager) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		if account.FrozenBalance.LessThan(amount) {
			return fmt.Errorf("frozen balance mismatch")
		}

		newAvailable := account.AvailableBalance.Add(amount)
		newFrozen := account.FrozenBalance.Sub(amount)
		return m.accountRepo.UpdateBalance(ctx, account.AccountID, account.Balance, newAvailable, newFrozen, account.Version)
	})
}

// --- Saga Distributed Transaction Support ---

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

		// 发布可靠结算出账事件
		return m.outbox.PublishInTx(ctx, ctx.Value("tx_db").(*gorm.DB), "account.balance.changed", fmt.Sprintf("SAGA-DED-%s", userID), map[string]any{
			"user_id": userID, "currency": currency, "change": amount.Neg().String(), "type": "SETTLE_OUT",
		})
	})
}

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

		// 发布可靠结算入账事件
		return m.outbox.PublishInTx(ctx, ctx.Value("tx_db").(*gorm.DB), "account.balance.changed", fmt.Sprintf("SAGA-ADD-%s", userID), map[string]any{
			"user_id": userID, "currency": currency, "change": amount.String(), "type": "SETTLE_IN",
		})
	})
}

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
