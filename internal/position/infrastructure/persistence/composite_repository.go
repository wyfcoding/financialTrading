package persistence

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

type compositePositionRepository struct {
	mysql domain.PositionRepository
	redis domain.PositionRepository
}

func NewCompositePositionRepository(mysql, redis domain.PositionRepository) domain.PositionRepository {
	return &compositePositionRepository{
		mysql: mysql,
		redis: redis,
	}
}

func (r *compositePositionRepository) Save(ctx context.Context, position *domain.Position) error {
	if err := r.mysql.Save(ctx, position); err != nil {
		return err
	}
	_ = r.redis.Save(ctx, position)
	return nil
}

func (r *compositePositionRepository) Get(ctx context.Context, id string) (*domain.Position, error) {
	// MySQL Get by ID (if needed, but composite primarily handles GetByUserSymbol)
	return r.mysql.Get(ctx, id)
}

func (r *compositePositionRepository) GetByUserSymbol(ctx context.Context, userID, symbol string) (*domain.Position, error) {
	// 1. Try Cache
	pos, err := r.redis.GetByUserSymbol(ctx, userID, symbol)
	if err == nil && pos != nil {
		return pos, nil
	}

	// 2. MySQL
	pos, err = r.mysql.GetByUserSymbol(ctx, userID, symbol)
	if err != nil || pos == nil {
		return pos, err
	}

	// 3. Fill Cache
	_ = r.redis.Save(ctx, pos)
	return pos, nil
}

func (r *compositePositionRepository) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Position, int64, error) {
	return r.mysql.GetByUser(ctx, userID, limit, offset)
}

func (r *compositePositionRepository) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*domain.Position, int64, error) {
	return r.mysql.GetBySymbol(ctx, symbol, limit, offset)
}

func (r *compositePositionRepository) Update(ctx context.Context, position *domain.Position) error {
	if err := r.mysql.Update(ctx, position); err != nil {
		return err
	}
	_ = r.redis.Save(ctx, position)
	return nil
}

func (r *compositePositionRepository) Close(ctx context.Context, id string, closePrice decimal.Decimal) error {
	if err := r.mysql.Close(ctx, id, closePrice); err != nil {
		return err
	}
	// Note: We don't have enough info here to update Redis key (userID:symbol)
	// Usually Close would be followed by a fetch or we invalidate all user positions in Redis
	return nil
}

func (r *compositePositionRepository) ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error {
	return r.mysql.ExecWithBarrier(ctx, barrier, fn)
}
