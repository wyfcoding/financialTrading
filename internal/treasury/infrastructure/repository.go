// Package infrastructure 资金服务基础设施层
package infrastructure

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormAccountRepository GORM 账户仓储实现
type GormAccountRepository struct {
	db *gorm.DB
}

// NewGormAccountRepository 创建账户仓储
func NewGormAccountRepository(db *gorm.DB) *GormAccountRepository {
	return &GormAccountRepository{db: db}
}

// Save 保存账户
func (r *GormAccountRepository) Save(ctx context.Context, account *domain.Account) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// GetByID 根据ID获取
func (r *GormAccountRepository) GetByID(ctx context.Context, id uint64) (*domain.Account, error) {
	var account domain.Account
	if err := r.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}
	return &account, nil
}

// GetByOwner 根据Owner获取
func (r *GormAccountRepository) GetByOwner(ctx context.Context, ownerID uint64, accType domain.AccountType, currency domain.Currency) (*domain.Account, error) {
	var account domain.Account
	err := r.db.WithContext(ctx).Where("owner_id = ? AND type = ? AND currency = ?", ownerID, accType, currency).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return &account, nil
}

// GetWithLock 悲观锁获取
func (r *GormAccountRepository) GetWithLock(ctx context.Context, id uint64) (*domain.Account, error) {
	var account domain.Account
	// SELECT * FROM accounts WHERE id = ? FOR UPDATE
	if err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, id).Error; err != nil {
		return nil, fmt.Errorf("account not found or lock failed: %w", err)
	}
	return &account, nil
}

// GormTransactionRepository GORM 交易流水仓储实现
type GormTransactionRepository struct {
	db *gorm.DB
}

// NewGormTransactionRepository 创建流水仓储
func NewGormTransactionRepository(db *gorm.DB) *GormTransactionRepository {
	return &GormTransactionRepository{db: db}
}

// Save 保存流水
func (r *GormTransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

// List 列出流水
func (r *GormTransactionRepository) List(ctx context.Context, accountID uint, txType *domain.TransactionType, limit, offset int) ([]*domain.Transaction, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Transaction{}).Where("account_id = ?", accountID)

	if txType != nil {
		query = query.Where("type = ?", *txType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var txs []*domain.Transaction
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&txs).Error; err != nil {
		return nil, 0, err
	}

	return txs, total, nil
}
