package persistence

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

type compositeAccountRepository struct {
	mysql domain.AccountRepository
	redis domain.AccountRepository
}

func NewCompositeAccountRepository(mysql, redis domain.AccountRepository) domain.AccountRepository {
	return &compositeAccountRepository{
		mysql: mysql,
		redis: redis,
	}
}

func (r *compositeAccountRepository) Save(ctx context.Context, account *domain.Account) error {
	if err := r.mysql.Save(ctx, account); err != nil {
		return err
	}
	_ = r.redis.Save(ctx, account) // 最佳实践：Cache 写入失败不影响主库
	return nil
}

func (r *compositeAccountRepository) Get(ctx context.Context, id string) (*domain.Account, error) {
	// 1. Try Cache
	acc, err := r.redis.Get(ctx, id)
	if err == nil && acc != nil {
		return acc, nil
	}

	// 2. Fallback to MySQL
	acc, err = r.mysql.Get(ctx, id)
	if err != nil || acc == nil {
		return acc, err
	}

	// 3. Backfill Cache
	_ = r.redis.Save(ctx, acc)
	return acc, nil
}

func (r *compositeAccountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Account, error) {
	// 列表查询通常不走 Redis（除非有专门的 Index Key）
	return r.mysql.GetByUserID(ctx, userID)
}

func (r *compositeAccountRepository) ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error {
	// Barrier 操作必须在主库上执行
	return r.mysql.ExecWithBarrier(ctx, barrier, fn)
}
