// 包 仓储实现
package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// AccountModel 账户数据库模型
// 对应数据库中的 accounts 表
type AccountModel struct {
	gorm.Model
	// 账户 ID，业务主键，唯一索引
	AccountID string `gorm:"column:account_id;type:varchar(50);uniqueIndex;not null" json:"account_id"`
	// 用户 ID，普通索引
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 账户类型
	AccountType string `gorm:"column:account_type;type:varchar(20);not null" json:"account_type"`
	// 货币
	Currency string `gorm:"column:currency;type:varchar(10);not null" json:"currency"`
	// 余额，高精度小数
	Balance string `gorm:"column:balance;type:decimal(20,8);not null" json:"balance"`
	// 可用余额，高精度小数
	AvailableBalance string `gorm:"column:available_balance;type:decimal(20,8);not null" json:"available_balance"`
	// 冻结余额，高精度小数
	FrozenBalance string `gorm:"column:frozen_balance;type:decimal(20,8);not null" json:"frozen_balance"`
}

// 指定表名
func (AccountModel) TableName() string {
	return "accounts"
}

// AccountRepositoryImpl 账户仓储实现
type AccountRepositoryImpl struct {
	db *gorm.DB
}

// NewAccountRepository 创建账户仓储
func NewAccountRepository(database *gorm.DB) domain.AccountRepository {
	return &AccountRepositoryImpl{
		db: database,
	}
}

// Save 保存账户
func (ar *AccountRepositoryImpl) Save(ctx context.Context, account *domain.Account) error {
	model := &AccountModel{
		Model:            account.Model,
		AccountID:        account.AccountID,
		UserID:           account.UserID,
		AccountType:      account.AccountType,
		Currency:         account.Currency,
		Balance:          account.Balance.String(),
		AvailableBalance: account.AvailableBalance.String(),
		FrozenBalance:    account.FrozenBalance.String(),
	}

	if err := ar.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save account",
			"account_id", account.AccountID,
			"error", err,
		)
		return fmt.Errorf("failed to save account: %w", err)
	}

	// 更新 domain 对象的 Model (例如 CreatedAt)
	account.Model = model.Model

	return nil
}

// Get 获取账户
func (ar *AccountRepositoryImpl) Get(ctx context.Context, accountID string) (*domain.Account, error) {
	var model AccountModel

	if err := ar.db.WithContext(ctx).Where("account_id = ?", accountID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get account",
			"account_id", accountID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return ar.modelToDomain(&model), nil
}

// GetByUser 获取用户账户
func (ar *AccountRepositoryImpl) GetByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	var models []AccountModel

	if err := ar.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get accounts by user",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get accounts by user: %w", err)
	}

	accounts := make([]*domain.Account, 0, len(models))
	for _, model := range models {
		accounts = append(accounts, ar.modelToDomain(&model))
	}

	return accounts, nil
}

// UpdateBalance 更新余额
func (ar *AccountRepositoryImpl) UpdateBalance(ctx context.Context, accountID string, balance, availableBalance, frozenBalance decimal.Decimal) error {
	if err := ar.db.WithContext(ctx).Model(&AccountModel{}).Where("account_id = ?", accountID).Updates(map[string]any{
		"balance":           balance.String(),
		"available_balance": availableBalance.String(),
		"frozen_balance":    frozenBalance.String(),
	}).Error; err != nil {
		logging.Error(ctx, "Failed to update balance",
			"account_id", accountID,
			"error", err,
		)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return nil
}

// 将数据库模型转换为领域对象
func (ar *AccountRepositoryImpl) modelToDomain(model *AccountModel) *domain.Account {
	balance, _ := decimal.NewFromString(model.Balance)
	availableBalance, _ := decimal.NewFromString(model.AvailableBalance)
	frozenBalance, _ := decimal.NewFromString(model.FrozenBalance)

	return &domain.Account{
		Model:            model.Model,
		AccountID:        model.AccountID,
		UserID:           model.UserID,
		AccountType:      model.AccountType,
		Currency:         model.Currency,
		Balance:          balance,
		AvailableBalance: availableBalance,
		FrozenBalance:    frozenBalance,
	}
}

// TransactionModel 交易记录数据库模型
type TransactionModel struct {
	gorm.Model
	// 交易 ID
	TransactionID string `gorm:"column:transaction_id;type:varchar(50);uniqueIndex;not null" json:"transaction_id"`
	// 账户 ID
	AccountID string `gorm:"column:account_id;type:varchar(50);index;not null" json:"account_id"`
	// 交易类型
	Type string `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// 金额
	Amount string `gorm:"column:amount;type:decimal(20,8);not null" json:"amount"`
	// 状态
	Status string `gorm:"column:status;type:varchar(20);not null" json:"status"`
}

// 指定表名
func (TransactionModel) TableName() string {
	return "transactions"
}

// TransactionRepositoryImpl 交易记录仓储实现
type TransactionRepositoryImpl struct {
	db *gorm.DB
}

// NewTransactionRepository 创建交易记录仓储
func NewTransactionRepository(database *gorm.DB) domain.TransactionRepository {
	return &TransactionRepositoryImpl{
		db: database,
	}
}

// Save 保存交易记录
func (tr *TransactionRepositoryImpl) Save(ctx context.Context, transaction *domain.Transaction) error {
	model := &TransactionModel{
		Model:         transaction.Model,
		TransactionID: transaction.TransactionID,
		AccountID:     transaction.AccountID,
		Type:          transaction.Type,
		Amount:        transaction.Amount.String(),
		Status:        transaction.Status,
	}

	if err := tr.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save transaction",
			"transaction_id", transaction.TransactionID,
			"error", err,
		)
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	// 更新 domain 对象的 Model
	transaction.Model = model.Model

	return nil
}

// GetHistory 获取交易历史
func (tr *TransactionRepositoryImpl) GetHistory(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	var models []TransactionModel
	var total int64

	query := tr.db.WithContext(ctx).Where("account_id = ?", accountID)

	if err := query.Model(&TransactionModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get transaction history",
			"account_id", accountID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get transaction history: %w", err)
	}

	transactions := make([]*domain.Transaction, 0, len(models))
	for _, model := range models {
		amount, _ := decimal.NewFromString(model.Amount)
		transactions = append(transactions, &domain.Transaction{
			Model:         model.Model,
			TransactionID: model.TransactionID,
			AccountID:     model.AccountID,
			Type:          model.Type,
			Amount:        amount,
			Status:        model.Status,
		})
	}

	return transactions, total, nil
}
