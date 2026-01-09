package application

import (
	"context"
	"fmt"
	"log/slog"

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

// CreateAccount 为指定用户创建特定币种的资产账户。
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
		slog.ErrorContext(ctx, "failed to create account", "user_id", req.UserID, "currency", req.Currency, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "account created", "account_id", accountID, "user_id", req.UserID, "currency", req.Currency)
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

// Deposit 执行资产充值，增加可用余额并产生财务流水。
func (m *AccountManager) Deposit(ctx context.Context, accountID string, amount decimal.Decimal) error {
	err := m.db.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		slog.ErrorContext(ctx, "deposit failed", "account_id", accountID, "amount", amount.String(), "error", err)
		return err
	}

	slog.InfoContext(ctx, "deposit successful", "account_id", accountID, "amount", amount.String())
	return nil
}

// FreezeBalance 冻结用户指定金额（例如下单时锁定保证金）。
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

	if err := m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen, account.Version); err != nil {
		slog.ErrorContext(ctx, "failed to freeze balance", "account_id", accountID, "amount", amount.String(), "error", err)
		return err
	}

	slog.InfoContext(ctx, "balance frozen", "account_id", accountID, "amount", amount.String(), "reason", reason)
	return nil
}

// UnfreezeBalance 解冻用户之前锁定的金额。
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

	if err := m.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen, account.Version); err != nil {
		slog.ErrorContext(ctx, "failed to unfreeze balance", "account_id", accountID, "amount", amount.String(), "error", err)
		return err
	}

	slog.InfoContext(ctx, "balance unfrozen", "account_id", accountID, "amount", amount.String())
	return nil
}

// DeductFrozenBalance 从已冻结的金额中执行真实扣除（例如成交后的资金结算）。
func (m *AccountManager) DeductFrozenBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	err := m.db.Transaction(func(tx *gorm.DB) error {
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
	if err != nil {
		slog.ErrorContext(ctx, "deduct frozen balance failed", "account_id", accountID, "amount", amount.String(), "error", err)
		return err
	}

	slog.InfoContext(ctx, "frozen balance deducted", "account_id", accountID, "amount", amount.String())
	return nil
}

// TccTryFreeze 执行 TCC 事务的第一阶段：尝试冻结资金。
func (m *AccountManager) TccTryFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	err := m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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
	if err != nil {
		slog.ErrorContext(ctx, "tcc_try_freeze failed", "user_id", userID, "currency", currency, "amount", amount.String(), "error", err)
		return err
	}
	slog.DebugContext(ctx, "tcc_try_freeze successful", "user_id", userID, "currency", currency, "amount", amount.String())
	return nil
}

// TccConfirmFreeze 执行 TCC 事务的第二阶段：确认冻结。
// 对于冻结操作，Confirm 阶段通常为幂等占位，因为资金已在 Try 阶段锁定。
func (m *AccountManager) TccConfirmFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error { return nil })
}

// TccCancelFreeze 执行 TCC 事务的取消阶段：释放已冻结的资金。
func (m *AccountManager) TccCancelFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	err := m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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
	if err != nil {
		slog.ErrorContext(ctx, "tcc_cancel_freeze failed", "user_id", userID, "currency", currency, "amount", amount.String(), "error", err)
		return err
	}
	slog.InfoContext(ctx, "tcc_cancel_freeze successful", "user_id", userID, "currency", currency, "amount", amount.String())
	return nil
}

// --- Saga Distributed Transaction Support ---

// SagaDeductFrozen 执行 Saga 事务中的正向扣款逻辑（针对已冻结资金）。
func (m *AccountManager) SagaDeductFrozen(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	err := m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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

		// 发布可靠结算出账事件，用于审计或下游对账
		return m.outbox.PublishInTx(ctx, ctx.Value("tx_db").(*gorm.DB), "account.balance.changed", fmt.Sprintf("SAGA-DED-%s", userID), map[string]any{
			"user_id": userID, "currency": currency, "change": amount.Neg().String(), "type": "SETTLE_OUT",
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "saga_deduct_frozen failed", "user_id", userID, "currency", currency, "amount", amount.String(), "error", err)
		return err
	}
	slog.InfoContext(ctx, "saga_deduct_frozen successful", "user_id", userID, "currency", currency, "amount", amount.String())
	return nil
}

// SagaRefundFrozen 执行 Saga 事务的补偿逻辑：将之前尝试扣除的金额退回冻结余额。
func (m *AccountManager) SagaRefundFrozen(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	err := m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		newBalance := account.Balance.Add(amount)
		newFrozen := account.FrozenBalance.Add(amount)
		return m.accountRepo.UpdateBalance(ctx, account.AccountID, newBalance, account.AvailableBalance, newFrozen, account.Version)
	})
	if err != nil {
		slog.ErrorContext(ctx, "saga_refund_frozen failed", "user_id", userID, "currency", currency, "amount", amount.String(), "error", err)
		return err
	}
	slog.InfoContext(ctx, "saga_refund_frozen successful", "user_id", userID, "currency", currency, "amount", amount.String())
	return nil
}

// SagaAddBalance 执行 Saga 事务中的正向加钱逻辑（增加总余额和可用余额）。
func (m *AccountManager) SagaAddBalance(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	err := m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		account, err := m.getAccount(ctx, userID, currency)
		if err != nil {
			return err
		}
		newBalance := account.Balance.Add(amount)
		newAvailable := account.AvailableBalance.Add(amount)

		if err := m.accountRepo.UpdateBalance(ctx, account.AccountID, newBalance, newAvailable, account.FrozenBalance, account.Version); err != nil {
			return err
		}

		return m.outbox.PublishInTx(ctx, ctx.Value("tx_db").(*gorm.DB), "account.balance.changed", fmt.Sprintf("SAGA-ADD-%s", userID), map[string]any{
			"user_id": userID, "currency": currency, "change": amount.String(), "type": "SETTLE_IN",
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "saga_add_balance failed", "user_id", userID, "currency", currency, "amount", amount.String(), "error", err)
		return err
	}
	slog.InfoContext(ctx, "saga_add_balance successful", "user_id", userID, "currency", currency, "amount", amount.String())
	return nil
}

// SagaSubBalance 执行 Saga 事务的补偿逻辑：扣除之前增加的余额。
func (m *AccountManager) SagaSubBalance(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	err := m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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
	if err != nil {
		slog.ErrorContext(ctx, "saga_sub_balance failed", "user_id", userID, "currency", currency, "amount", amount.String(), "error", err)
		return err
	}
	slog.InfoContext(ctx, "saga_sub_balance successful", "user_id", userID, "currency", currency, "amount", amount.String())
	return nil
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
