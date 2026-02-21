package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type matchingRepository struct {
	// 实战中通常使用 Redis 配合本地内存快照
}

func NewMatchingRepository() domain.OrderBookRepository {
	return &matchingRepository{}
}

func (r *matchingRepository) SaveSnapshot(_ context.Context, _ *domain.OrderBookSnapshot) error {
	return nil
}
