// 包 账户服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// CreateAccountRequest 创建账户请求 DTO
// 用于接收创建账户的请求参数
type CreateAccountRequest struct {
	UserID      string // 用户 ID
	AccountType string // 账户类型（如 SPOT, MARGIN）
	Currency    string // 币种（如 USD, BTC）
}

// AccountDTO 账户 DTO
// 用于向外层返回账户详情
type AccountDTO struct {
	AccountID        string // 账户 ID
	UserID           string // 用户 ID
	AccountType      string // 账户类型
	Currency         string // 币种
	Balance          string // 总余额
	AvailableBalance string // 可用余额
	FrozenBalance    string // 冻结余额
	CreatedAt        int64  // 创建时间戳（秒）
	UpdatedAt        int64  // 更新时间戳（秒）
}

// AccountApplicationService 账户应用服务
type AccountApplicationService struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
}

// NewAccountApplicationService 创建账户应用服务
func NewAccountApplicationService(accountRepo domain.AccountRepository, transactionRepo domain.TransactionRepository) *AccountApplicationService {
	return &AccountApplicationService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// CreateAccount 创建账户
// 用例流程：
// 1. 验证输入参数
// 2. 生成账户 ID
// 3. 创建账户对象
// 4. 保存到仓储
func (aas *AccountApplicationService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountDTO, error) {
	logging.Info(ctx, "Creating new account",
		"user_id", req.UserID,
		"account_type", req.AccountType,
		"currency", req.Currency,
	)

	// 验证输入
	if req.UserID == "" || req.AccountType == "" || req.Currency == "" {
		logging.Warn(ctx, "Invalid account creation parameters",
			"user_id", req.UserID,
			"account_type", req.AccountType,
			"currency", req.Currency,
		)
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 生成账户 ID
	accountID := fmt.Sprintf("ACC-%d", idgen.GenID())

	// 创建账户对象
	account := &domain.Account{
		AccountID:        accountID,
		UserID:           req.UserID,
		AccountType:      req.AccountType,
		Currency:         req.Currency,
		Balance:          decimal.Zero,
		AvailableBalance: decimal.Zero,
		FrozenBalance:    decimal.Zero,
	}

	// 保存到仓储
	if err := aas.accountRepo.Save(ctx, account); err != nil {
		logging.Error(ctx, "Failed to save account to repository",
			"account_id", accountID,
			"user_id", req.UserID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	logging.Info(ctx, "Account created successfully",
		"account_id", accountID,
		"user_id", req.UserID,
		"account_type", req.AccountType,
		"currency", req.Currency,
	)

	// 转换为 DTO
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

// GetAccount 获取账户信息
func (aas *AccountApplicationService) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	// 验证输入
	if accountID == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	// 获取账户
	account, err := aas.accountRepo.Get(ctx, accountID)
	if err != nil {
		logging.Error(ctx, "Failed to get account",
			"account_id", accountID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	// 转换为 DTO
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
// 用例流程：
// 1. 验证账户存在
// 2. 更新余额
// 3. 创建交易记录
func (aas *AccountApplicationService) Deposit(ctx context.Context, accountID string, amount decimal.Decimal) error {
	logging.Info(ctx, "Processing deposit",
		"account_id", accountID,
		"amount", amount.String(),
	)

	// 验证输入
	if accountID == "" || amount.LessThanOrEqual(decimal.Zero) {
		logging.Warn(ctx, "Invalid deposit parameters",
			"account_id", accountID,
			"amount", amount.String(),
		)
		return fmt.Errorf("invalid request parameters")
	}

	// 获取账户
	account, err := aas.accountRepo.Get(ctx, accountID)
	if err != nil {
		logging.Error(ctx, "Failed to retrieve account for deposit",
			"account_id", accountID,
			"error", err,
		)
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		logging.Warn(ctx, "Account not found for deposit",
			"account_id", accountID,
		)
		return fmt.Errorf("account not found: %s", accountID)
	}

	// 更新余额
	newBalance := account.Balance.Add(amount)
	newAvailableBalance := account.AvailableBalance.Add(amount)

	if err := aas.accountRepo.UpdateBalance(ctx, accountID, newBalance, newAvailableBalance, account.FrozenBalance); err != nil {
		logging.Error(ctx, "Failed to update balance for deposit",
			"account_id", accountID,
			"old_balance", account.Balance.String(),
			"new_balance", newBalance.String(),
			"error", err,
		)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// 创建交易记录
	transaction := &domain.Transaction{
		TransactionID: fmt.Sprintf("TXN-%d", idgen.GenID()),
		AccountID:     accountID,
		Type:          "DEPOSIT",
		Amount:        amount,
		Status:        "COMPLETED",
	}

	if err := aas.transactionRepo.Save(ctx, transaction); err != nil {
		logging.Error(ctx, "Failed to save deposit transaction",
			"transaction_id", transaction.TransactionID,
			"account_id", accountID,
			"error", err,
		)
	}

	logging.Info(ctx, "Deposit completed successfully",
		"account_id", accountID,
		"amount", amount.String(),
		"new_balance", newBalance.String(),
		"transaction_id", transaction.TransactionID,
	)

	return nil
}

// FreezeBalance 冻结余额 (用于分布式事务第一阶段)
func (aas *AccountApplicationService) FreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal, reason string) error {
	logging.Info(ctx, "Freezing balance", "account_id", accountID, "amount", amount.String(), "reason", reason)

	account, err := aas.accountRepo.Get(ctx, accountID)
	if err != nil {
		logging.Error(ctx, "Failed to get account for freeze", "account_id", accountID, "error", err)
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		logging.Warn(ctx, "Account not found for freeze", "account_id", accountID)
		return fmt.Errorf("account not found")
	}

	if account.AvailableBalance.LessThan(amount) {
		logging.Warn(ctx, "Insufficient available balance for freeze",
			"account_id", accountID,
			"available", account.AvailableBalance.String(),
			"required", amount.String(),
		)
		return fmt.Errorf("insufficient available balance")
	}

	newAvailable := account.AvailableBalance.Sub(amount)
	newFrozen := account.FrozenBalance.Add(amount)

	if err := aas.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen); err != nil {
		logging.Error(ctx, "Failed to update balance for freeze",
			"account_id", accountID,
			"error", err,
		)
		return fmt.Errorf("failed to update frozen balance: %w", err)
	}

	logging.Info(ctx, "Balance frozen successfully", "account_id", accountID, "amount", amount.String())
	return nil
}

// UnfreezeBalance 解冻余额 (用于分布式事务回滚)
func (aas *AccountApplicationService) UnfreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	logging.Info(ctx, "Unfreezing balance", "account_id", accountID, "amount", amount.String())

	account, err := aas.accountRepo.Get(ctx, accountID)
	if err != nil {
		logging.Error(ctx, "Failed to get account for unfreeze", "account_id", accountID, "error", err)
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		logging.Warn(ctx, "Account not found for unfreeze", "account_id", accountID)
		return fmt.Errorf("account not found")
	}

	if account.FrozenBalance.LessThan(amount) {
		logging.Warn(ctx, "Frozen balance insufficient to unfreeze",
			"account_id", accountID,
			"frozen", account.FrozenBalance.String(),
			"required", amount.String(),
		)
		return fmt.Errorf("frozen balance insufficient to unfreeze")
	}

	newAvailable := account.AvailableBalance.Add(amount)
	newFrozen := account.FrozenBalance.Sub(amount)

	if err := aas.accountRepo.UpdateBalance(ctx, accountID, account.Balance, newAvailable, newFrozen); err != nil {
		logging.Error(ctx, "Failed to update balance for unfreeze", "account_id", accountID, "error", err)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	logging.Info(ctx, "Balance unfrozen successfully", "account_id", accountID, "amount", amount.String())
	return nil
}

// DeductFrozenBalance 扣除已冻结的余额 (用于分布式事务确认)
func (aas *AccountApplicationService) DeductFrozenBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	logging.Info(ctx, "Deducting frozen balance", "account_id", accountID, "amount", amount.String())

	account, err := aas.accountRepo.Get(ctx, accountID)
	if err != nil {
		logging.Error(ctx, "Failed to get account for deduct", "account_id", accountID, "error", err)
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		logging.Warn(ctx, "Account not found for deduct", "account_id", accountID)
		return fmt.Errorf("account not found")
	}

	if account.FrozenBalance.LessThan(amount) {
		logging.Warn(ctx, "Frozen balance insufficient to deduct",
			"account_id", accountID,
			"frozen", account.FrozenBalance.String(),
			"required", amount.String(),
		)
		return fmt.Errorf("frozen balance insufficient to deduct")
	}

	newBalance := account.Balance.Sub(amount)
	newFrozen := account.FrozenBalance.Sub(amount)

	if err := aas.accountRepo.UpdateBalance(ctx, accountID, newBalance, account.AvailableBalance, newFrozen); err != nil {
		logging.Error(ctx, "Failed to update balance for deduct", "account_id", accountID, "error", err)
		return err
	}

	// 记录真实扣款流水
	transaction := &domain.Transaction{
		TransactionID: fmt.Sprintf("DED-%d", idgen.GenID()),
		AccountID:     accountID,
		Type:          "DEDUCT",
		Amount:        amount,
		Status:        "COMPLETED",
	}
	if err := aas.transactionRepo.Save(ctx, transaction); err != nil {
		logging.Error(ctx, "Failed to save deduct transaction", "account_id", accountID, "error", err)
	}

	logging.Info(ctx, "Frozen balance deducted successfully",
		"account_id", accountID,
		"amount", amount.String(),
		"transaction_id", transaction.TransactionID,
	)
	return nil
}
