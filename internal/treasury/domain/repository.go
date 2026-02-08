// Package domain 资金服务仓储接口
package domain

import "context"

type AccountRepository interface {
	Save(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id uint64) (*Account, error)
	GetByOwner(ctx context.Context, ownerID uint64, accType AccountType, currency Currency) (*Account, error)
	GetWithLock(ctx context.Context, id uint64) (*Account, error) // 悲观锁获取
}

type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	List(ctx context.Context, accountID uint, txType *TransactionType, limit, offset int) ([]*Transaction, int64, error)
}
