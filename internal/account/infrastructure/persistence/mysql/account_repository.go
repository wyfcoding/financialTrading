// Package mysql 提供了账户仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/dtm"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AccountModel 账户数据库模型
type AccountModel struct {
	gorm.Model
	AccountID        string `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null"`
	UserID           string `gorm:"column:user_id;type:varchar(32);index;not null"`
	AccountType      string `gorm:"column:account_type;type:varchar(20);not null"`
	Currency         string `gorm:"column:currency;type:varchar(10);not null"`
	Balance          string `gorm:"column:balance;type:decimal(32,18);default:'0';not null"`
	AvailableBalance string `gorm:"column:available_balance;type:decimal(32,18);default:'0';not null"`
	FrozenBalance    string `gorm:"column:frozen_balance;type:decimal(32,18);default:'0';not null"`
	Version          int64  `gorm:"column:version;default:0;not null"`
}

// TableName 指定表名
func (AccountModel) TableName() string {
	return "accounts"
}

// accountRepositoryImpl 是 domain.AccountRepository 接口的 GORM 实现。
type accountRepositoryImpl struct {
	db *gorm.DB
}

// NewAccountRepository 创建账户仓储实例
func NewAccountRepository(db *gorm.DB) domain.AccountRepository {
	return &accountRepositoryImpl{
		db: db,
	}
}

// Save 实现 domain.AccountRepository.Save
func (r *accountRepositoryImpl) Save(ctx context.Context, account *domain.Account) error {
	model := &AccountModel{
		Model:            account.Model,
		AccountID:        account.AccountID,
		UserID:           account.UserID,
		AccountType:      account.AccountType,
		Currency:         account.Currency,
		Balance:          account.Balance.String(),
		AvailableBalance: account.AvailableBalance.String(),
		FrozenBalance:    account.FrozenBalance.String(),
		Version:          account.Version,
	}

	err := r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "account_type", "currency", "balance", "available_balance", "frozen_balance"}),
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "account_repository.Save failed", "account_id", account.AccountID, "error", err)
		return fmt.Errorf("failed to save account: %w", err)
	}

	account.Model = model.Model
	return nil
}

// Get 实现 domain.AccountRepository.Get
func (r *accountRepositoryImpl) Get(ctx context.Context, accountID string) (*domain.Account, error) {
	var model AccountModel
	if err := r.getDB(ctx).Where("account_id = ?", accountID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "account_repository.Get failed", "account_id", accountID, "error", err)
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return r.toDomain(&model), nil
}

// GetByUser 实现 domain.AccountRepository.GetByUser
func (r *accountRepositoryImpl) GetByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	var models []AccountModel
	if err := r.getDB(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		logging.Error(ctx, "account_repository.GetByUser failed", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get accounts by user: %w", err)
	}

	accounts := make([]*domain.Account, len(models))
	for i, m := range models {
		accounts[i] = r.toDomain(&m)
	}
	return accounts, nil
}

// UpdateBalance 实现 domain.AccountRepository.UpdateBalance (带乐观锁)
func (r *accountRepositoryImpl) UpdateBalance(ctx context.Context, accountID string, balance, available, frozen decimal.Decimal, currentVersion int64) error {
	result := r.getDB(ctx).Model(&AccountModel{}).
		Where("account_id = ? AND version = ?", accountID, currentVersion).
		Updates(map[string]interface{}{
			"balance":           balance.String(),
			"available_balance": available.String(),
			"frozen_balance":    frozen.String(),
			"version":           currentVersion + 1, // 版本号递增
		})

	if result.Error != nil {
		logging.Error(ctx, "account_repository.UpdateBalance failed", "account_id", accountID, "error", result.Error)
		return fmt.Errorf("failed to update balance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		// 可能是账户不存在，也可能是版本号已被其他事务修改
		return domain.ErrConcurrentUpdate
	}

	return nil
}

// ExecWithBarrier 实现 domain.AccountRepository.ExecWithBarrier
func (r *accountRepositoryImpl) ExecWithBarrier(ctx context.Context, barrier interface{}, fn func(ctx context.Context) error) error {
	// 使用 pkg/dtm 的 CallWithGorm 辅助函数 (通过反射避免直接依赖 dtmcli)
	return dtm.CallWithGorm(ctx, barrier, r.db, func(tx *gorm.DB) error {
		// 将事务 DB 注入 context，以便后续操作使用该事务
		txCtx := context.WithValue(ctx, "tx_db", tx)
		return fn(txCtx)
	})
}

// getDBFromContext 尝试从 Context 获取事务 DB，否则返回默认 DB
func (r *accountRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx_db").(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *accountRepositoryImpl) toDomain(m *AccountModel) *domain.Account {
	balance, err := decimal.NewFromString(m.Balance)
	if err != nil {
		balance = decimal.Zero
	}
	available, err := decimal.NewFromString(m.AvailableBalance)
	if err != nil {
		available = decimal.Zero
	}
	frozen, err := decimal.NewFromString(m.FrozenBalance)
	if err != nil {
		frozen = decimal.Zero
	}

	return &domain.Account{
		Model:            m.Model,
		AccountID:        m.AccountID,
		UserID:           m.UserID,
		AccountType:      m.AccountType,
		Currency:         m.Currency,
		Balance:          balance,
		AvailableBalance: available,
		FrozenBalance:    frozen,
		Version:          m.Version,
	}
}

// TransactionModel 交易记录数据库模型
type TransactionModel struct {
	gorm.Model
	TransactionID string `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null"`
	AccountID     string `gorm:"column:account_id;type:varchar(32);index;not null"`
	Type          string `gorm:"column:type;type:varchar(20);not null"`
	Amount        string `gorm:"column:amount;type:decimal(32,18);not null"`
	Status        string `gorm:"column:status;type:varchar(20);not null"`
}

// TableName 指定表名
func (TransactionModel) TableName() string {
	return "transactions"
}

// transactionRepositoryImpl 是 domain.TransactionRepository 接口的 GORM 实现。
type transactionRepositoryImpl struct {
	db *gorm.DB
}

// NewTransactionRepository 创建交易记录仓储实例
func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepositoryImpl{db: db}
}

// Save 实现 domain.TransactionRepository.Save
func (r *transactionRepositoryImpl) Save(ctx context.Context, transaction *domain.Transaction) error {
	model := &TransactionModel{
		Model:         transaction.Model,
		TransactionID: transaction.TransactionID,
		AccountID:     transaction.AccountID,
		Type:          transaction.Type,
		Amount:        transaction.Amount.String(),
		Status:        transaction.Status,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "transaction_repository.Save failed", "transaction_id", transaction.TransactionID, "error", err)
		return fmt.Errorf("failed to save transaction: %w", err)
	}
	transaction.Model = model.Model
	return nil
}

// GetHistory 实现 domain.TransactionRepository.GetHistory
func (r *transactionRepositoryImpl) GetHistory(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	var models []TransactionModel
	var total int64
	db := r.db.WithContext(ctx).Model(&TransactionModel{}).Where("account_id = ?", accountID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "transaction_repository.GetHistory failed", "account_id", accountID, "error", err)
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}

	txs := make([]*domain.Transaction, len(models))
	for i, m := range models {
		amount, err := decimal.NewFromString(m.Amount)
		if err != nil {
			amount = decimal.Zero
		}
		txs[i] = &domain.Transaction{
			Model:         m.Model,
			TransactionID: m.TransactionID,
			AccountID:     m.AccountID,
			Type:          m.Type,
			Amount:        amount,
			Status:        m.Status,
		}
	}
	return txs, total, nil
}
