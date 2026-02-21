// 变更说明：
// 撮合引擎快照仓储机制。
// 由于撮合引擎是在全内存中运算的，如果机器断电会丢失整个订单簿 (OrderBook)。
// 必须通过独立协程定期 (如每10秒) 将全内存状态序列化为快照打入 Redis 或 DB。
package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type SnapshotRepositoryImpl struct {
	rdb redis.UniversalClient
}

func NewSnapshotRepository(rdb redis.UniversalClient) *SnapshotRepositoryImpl {
	return &SnapshotRepositoryImpl{rdb: rdb}
}

// SaveSnapshot 将订单薄内存导出为快照落入高速 Redis。
func (r *SnapshotRepositoryImpl) SaveSnapshot(ctx context.Context, symbol string, book *domain.FastOrderBook) error {
	data, err := json.Marshal(book)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("matching:snapshot:%s", symbol)
	// 保留最新的快照即可，崩溃重启时从最新快照 + Kafka CDC 流水回放恢复状态
	return r.rdb.Set(ctx, key, data, 24*time.Hour).Err()
}

// LoadSnapshot 在引擎重启时拉取快照。
func (r *SnapshotRepositoryImpl) LoadSnapshot(ctx context.Context, symbol string) (*domain.FastOrderBook, error) {
	key := fmt.Sprintf("matching:snapshot:%s", symbol)
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 没有历史快照，纯净启动
		}
		return nil, err
	}

	var book domain.FastOrderBook
	if err := json.Unmarshal([]byte(val), &book); err != nil {
		return nil, err
	}
	return &book, nil
}
