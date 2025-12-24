// Package persistence 提供了仓储接口的具体实现。
// 这一层是基础设施层的一部分，负责将领域对象与具体的持久化技术（如数据库）进行映射和交互。
package persistence

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// accountRepositoryImpl 是 domain.AccountRepository 接口的 GORM 实现。
// 它依赖一个 *gorm.DB 连接实例。
type accountRepositoryImpl struct {
	db *gorm.DB
}

// NewAccountRepository 是 accountRepositoryImpl 的构造函数。
func NewAccountRepository(db *gorm.DB) domain.AccountRepository {
	return &accountRepositoryImpl{
		db: db,
	}
}

// toDomainAccount 将数据库模型 `domain.Account` 转换为领域实体 `domain.Account`。
// 这是一个关键的映射步骤，用于隔离数据库结构和领域逻辑。
func (r *accountRepositoryImpl) toDomainAccount(model *domain.Account) *domain.Account {
	// 在实际应用中，如果数据库模型和领域模型结构差异较大，这里会包含更多的转换逻辑。
	// 当前结构类似，主要是 decimal 类型的转换。
	return model
}

// fromDomainAccount 将领域实体 `domain.Account` 转换为可供 GORM 操作的数据库模型。
func (r *accountRepositoryImpl) fromDomainAccount(entity *domain.Account) *domain.Account {
	// 这里同样，当前实现较为直接。
	return entity
}

// Save 实现了 domain.AccountRepository.Save 方法。
// 使用 `clauses.OnConflict` 来实现 "Upsert" (Update or Insert) 逻辑。
func (r *accountRepositoryImpl) Save(ctx context.Context, account *domain.Account) error {
	dbModel := r.fromDomainAccount(account)

	// 冲突处理: 如果 account_id 已存在，则更新所有字段。
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "account_type", "currency", "balance", "available_balance", "frozen_balance"}),
	}).Create(dbModel).Error; err != nil {
		logging.Error(ctx, "Failed to save account to DB", "account_id", account.AccountID, "error", err)
		return err
	}

	// GORM 创建后会自动填充主键等信息，这里可以更新回原实体。
	*account = *r.toDomainAccount(dbModel)
	return nil
}

// Get 实现了 domain.AccountRepository.Get 方法。
func (r *accountRepositoryImpl) Get(ctx context.Context, accountID string) (*domain.Account, error) {
	var model domain.Account

	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err // 返回 gorm 的标准未找到错误，以便上层可以判断
		}
		logging.Error(ctx, "Failed to get account from DB", "account_id", accountID, "error", err)
		return nil, err
	}

	return r.toDomainAccount(&model), nil
}

// GetByUser 实现了 domain.AccountRepository.GetByUser 方法。
func (r *accountRepositoryImpl) GetByUser(ctx context.Context, userID string) ([]*domain.Account, error) {
	var models []*domain.Account

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get accounts by user from DB", "user_id", userID, "error", err)
		return nil, err
	}

	entities := make([]*domain.Account, len(models))
	for i, model := range models {
		entities[i] = r.toDomainAccount(model)
	}

	return entities, nil
}

// UpdateBalance 实现了 domain.AccountRepository.UpdateBalance 方法。
// 它使用 `clauses.Locking{Strength: "UPDATE"}` 来获取行锁，确保并发更新的原子性和一致性。
func (r *accountRepositoryImpl) UpdateBalance(ctx context.Context, accountID string, newBalance, newAvailable, newFrozen decimal.Decimal) error {
	result := r.db.WithContext(ctx).Model(&domain.Account{}).Where("account_id = ?", accountID).Updates(map[string]any{
		"balance":           newBalance,
		"available_balance": newAvailable,
		"frozen_balance":    newFrozen,
	})

	if result.Error != nil {
		logging.Error(ctx, "Failed to update account balance in DB", "account_id", accountID, "error", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // 如果没有行被更新，说明记录可能不存在
	}
	return nil
}

// --- TransactionRepository ---

// transactionRepositoryImpl 是 domain.TransactionRepository 接口的 GORM 实现。
type transactionRepositoryImpl struct {
	db *gorm.DB
}

// NewTransactionRepository 是 transactionRepositoryImpl 的构造函数。
func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepositoryImpl{db: db}
}

// Save 实现了 domain.TransactionRepository.Save 方法。
func (r *transactionRepositoryImpl) Save(ctx context.Context, transaction *domain.Transaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

// GetHistory 实现了 domain.TransactionRepository.GetHistory 方法。
func (r *transactionRepositoryImpl) GetHistory(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	var transactions []*domain.Transaction
	var total int64

	db := r.db.WithContext(ctx).Model(&domain.Transaction{}).Where("account_id = ?", accountID)

	// 首先计算总数，用于分页
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 然后获取当前页的数据
	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
