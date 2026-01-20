package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Save(ctx context.Context, account *domain.Account) error {
	po := &AccountPO{}
	po.FromDomain(account)

	db := r.getDB(ctx)

	// 使用 Upsert (On Duplicate Key Update)
	// 注意：这里简单的 Save 可能不够原子，如果涉及到版本控制 (Optimistic Lock)
	// 应该使用 Updates + Version Check
	if account.Version == 0 {
		return db.Create(po).Error
	}

	// 乐观锁更新
	result := db.Model(&AccountPO{}).
		Where("account_id = ? AND version = ?", account.ID, account.Version).
		Updates(map[string]interface{}{
			"balance":           account.Balance,
			"available_balance": account.AvailableBalance,
			"frozen_balance":    account.FrozenBalance,
			"version":           account.Version + 1, // 版本号自增
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock failed: account modified by another transaction")
	}

	// 更新内存中的版本号，以便后续使用
	account.Version++
	return nil
}

func (r *AccountRepository) Get(ctx context.Context, id string) (*domain.Account, error) {
	var po AccountPO
	if err := r.getDB(ctx).Where("account_id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.ToDomain(), nil
}

func (r *AccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Account, error) {
	var pos []AccountPO
	if err := r.getDB(ctx).Where("user_id = ?", userID).Find(&pos).Error; err != nil {
		return nil, err
	}

	accounts := make([]*domain.Account, len(pos))
	for i, po := range pos {
		accounts[i] = po.ToDomain()
	}
	return accounts, nil
}

// ExecWithBarrier 用于支持 DTM 等 saga 模式的 barrier
func (r *AccountRepository) ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error {
	// 这是一个简化的适配，实际应该调用 dtmcli.BarrierFromContext(ctx).Call(...)
	// 由于 domain 层不应该依赖 dtmcli，这里假设 barrier 是传递进来的业务参数或 nil
	// 这里简单开启事务
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *AccountRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
