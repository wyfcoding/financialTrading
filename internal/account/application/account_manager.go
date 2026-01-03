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

// --- TCC Distributed Transaction Support ---

// TccTryFreeze TCC Try: 预冻结资金
func (m *AccountManager) TccTryFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 1. 查找账户
		accounts, err := m.accountRepo.GetByUser(ctx, userID)
		if err != nil {
			return err
		}
		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		// 2. 检查余额
		if targetAccount.AvailableBalance.LessThan(amount) {
			return fmt.Errorf("insufficient available balance")
		}

		// 3. 执行冻结 (Available -> Frozen)
		newAvailable := targetAccount.AvailableBalance.Sub(amount)
		newFrozen := targetAccount.FrozenBalance.Add(amount)

		// 注意：ExecWithBarrier 内部注入了带事务的 DB 到 ctx，所以 UpdateBalance 会在事务中执行
		return m.accountRepo.UpdateBalance(ctx, targetAccount.AccountID, targetAccount.Balance, newAvailable, newFrozen)
	})
}

// TccConfirmFreeze TCC Confirm: 确认冻结 (Try 阶段已完成资金划转，此处为空操作，仅满足协议)
func (m *AccountManager) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// Try 阶段已经将资金移动到冻结状态，确认阶段不需要做额外操作
		// 除非业务需要修改状态（例如生成一条“确认冻结”的流水）
		return nil
	})
}

// TccCancelFreeze TCC Cancel: 取消冻结 (Frozen -> Available)
func (m *AccountManager) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return m.accountRepo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 1. 查找账户
		accounts, err := m.accountRepo.GetByUser(ctx, userID)
		if err != nil {
			return err
		}
		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		// 2. 执行解冻 (Frozen -> Available)
		// 注意：这里假设 Try 成功执行了。如果 Try 没执行，DTM 的空补偿策略会处理（Barrier 自动处理）
		// 如果是“悬挂”请求（Cancel 先到），Barrier 也会拦截
		
		// 只有当 Frozen 足够时才解冻。
		// 在 TCC Cancel 中，理论上资金一定在 Frozen 中（因为 Try 成功了）。
		// 但为了健壮性，还是检查一下。
		if targetAccount.FrozenBalance.LessThan(amount) {
			// 这通常不应该发生，除非手动干预了数据
			return fmt.Errorf("frozen balance mismatch during cancel")
		}

		newAvailable := targetAccount.AvailableBalance.Add(amount)
		newFrozen := targetAccount.FrozenBalance.Sub(amount)

		return m.accountRepo.UpdateBalance(ctx, targetAccount.AccountID, targetAccount.Balance, newAvailable, newFrozen)
	})
}
