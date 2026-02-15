// Package domain 资金服务仓储接口
package domain

import (
	"context"
	"time"
)

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

type CashPoolRepository interface {
	Save(ctx context.Context, pool *CashPool) error
	GetByID(ctx context.Context, id uint64) (*CashPool, error)
	ListAll(ctx context.Context) ([]*CashPool, error)
}

type LiquidityForecastRepository interface {
	Save(ctx context.Context, forecast *LiquidityForecast) error
	ListByPoolAndDateRange(ctx context.Context, poolID uint64, start, end time.Time) ([]*LiquidityForecast, error)
}

type TransferInstructionRepository interface {
	Save(ctx context.Context, instruction *TransferInstruction) error
	GetByID(ctx context.Context, id string) (*TransferInstruction, error)
	ListPending(ctx context.Context, limit int) ([]*TransferInstruction, error)
}
