//go:build settlement_experimental
// +build settlement_experimental

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

// RedisSettlementRepository Redis实现的结算仓储
type RedisSettlementRepository struct {
	client *redis.Client
	prefix string
}

// NewRedisSettlementRepository 创建Redis结算仓储
func NewRedisSettlementRepository(client *redis.Client) *RedisSettlementRepository {
	return &RedisSettlementRepository{
		client: client,
		prefix: "settlement:",
	}
}

// SaveInstruction 保存结算指令
func (r *RedisSettlementRepository) SaveInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	key := r.getInstructionKey(instruction.ID)

	// 序列化指令
	data, err := json.Marshal(instruction)
	if err != nil {
		return fmt.Errorf("failed to marshal instruction: %w", err)
	}

	// 保存到Redis
	err = r.client.Set(ctx, key, data, 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save instruction to Redis: %w", err)
	}

	// 添加到索引
	err = r.addToIndex(ctx, "instruction", instruction.ID, instruction.SettlementDate)
	if err != nil {
		return fmt.Errorf("failed to add to index: %w", err)
	}

	return nil
}

// GetInstruction 获取结算指令
func (r *RedisSettlementRepository) GetInstruction(ctx context.Context, id string) (*domain.SettlementInstruction, error) {
	key := r.getInstructionKey(id)

	// 从Redis获取
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("instruction not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get instruction from Redis: %w", err)
	}

	// 反序列化
	var instruction domain.SettlementInstruction
	err = json.Unmarshal(data, &instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal instruction: %w", err)
	}

	return &instruction, nil
}

// GetInstructionByRef 通过引用获取结算指令
func (r *RedisSettlementRepository) GetInstructionByRef(ctx context.Context, ref string) (*domain.SettlementInstruction, error) {
	// 创建引用索引键
	refKey := r.getRefIndexKey(ref)

	// 获取指令ID
	instructionID, err := r.client.Get(ctx, refKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("instruction not found for ref: %s", ref)
		}
		return nil, fmt.Errorf("failed to get instruction ID from Redis: %w", err)
	}

	// 获取指令
	return r.GetInstruction(ctx, instructionID)
}

// GetInstructionsByTrade 通过交易获取结算指令
func (r *RedisSettlementRepository) GetInstructionsByTrade(ctx context.Context, tradeID string) ([]*domain.SettlementInstruction, error) {
	// 创建交易索引键
	tradeKey := r.getTradeIndexKey(tradeID)

	// 获取指令ID列表
	instructionIDs, err := r.client.SMembers(ctx, tradeKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instruction IDs from Redis: %w", err)
	}

	var instructions []*domain.SettlementInstruction
	for _, id := range instructionIDs {
		instruction, err := r.GetInstruction(ctx, id)
		if err != nil {
			// 记录错误但继续处理其他指令
			fmt.Printf("Failed to get instruction %s: %v\n", id, err)
			continue
		}
		instructions = append(instructions, instruction)
	}

	return instructions, nil
}

// GetInstructionsByBatch 通过批次获取结算指令
func (r *RedisSettlementRepository) GetInstructionsByBatch(ctx context.Context, batchID string) ([]*domain.SettlementInstruction, error) {
	// 创建批次索引键
	batchKey := r.getBatchIndexKey(batchID)

	// 获取指令ID列表
	instructionIDs, err := r.client.SMembers(ctx, batchKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instruction IDs from Redis: %w", err)
	}

	var instructions []*domain.SettlementInstruction
	for _, id := range instructionIDs {
		instruction, err := r.GetInstruction(ctx, id)
		if err != nil {
			// 记录错误但继续处理其他指令
			fmt.Printf("Failed to get instruction %s: %v\n", id, err)
			continue
		}
		instructions = append(instructions, instruction)
	}

	return instructions, nil
}

// GetPendingInstructionsByDate 获取指定日期的待结算指令
func (r *RedisSettlementRepository) GetPendingInstructionsByDate(ctx context.Context, settlementDate time.Time) ([]*domain.SettlementInstruction, error) {
	// 创建日期索引键
	dateKey := r.getDateIndexKey(settlementDate)

	// 获取指令ID列表
	instructionIDs, err := r.client.SMembers(ctx, dateKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get instruction IDs from Redis: %w", err)
	}

	var instructions []*domain.SettlementInstruction
	for _, id := range instructionIDs {
		instruction, err := r.GetInstruction(ctx, id)
		if err != nil {
			// 记录错误但继续处理其他指令
			fmt.Printf("Failed to get instruction %s: %v\n", id, err)
			continue
		}

		// 只返回待结算的指令
		if instruction.Status == domain.SettlementPending {
			instructions = append(instructions, instruction)
		}
	}

	return instructions, nil
}

// UpdateInstruction 更新结算指令
func (r *RedisSettlementRepository) UpdateInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 先获取现有指令
	existing, err := r.GetInstruction(ctx, instruction.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing instruction: %w", err)
	}

	// 检查状态变化
	if existing.Status != instruction.Status {
		// 更新状态索引
		err = r.updateStatusIndex(ctx, existing, instruction)
		if err != nil {
			return fmt.Errorf("failed to update status index: %w", err)
		}
	}

	// 保存更新后的指令
	return r.SaveInstruction(ctx, instruction)
}

// DeleteInstruction 删除结算指令
func (r *RedisSettlementRepository) DeleteInstruction(ctx context.Context, id string) error {
	// 获取指令
	instruction, err := r.GetInstruction(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get instruction: %w", err)
	}

	// 从索引中移除
	err = r.removeFromIndex(ctx, "instruction", instruction.ID, instruction.SettlementDate)
	if err != nil {
		return fmt.Errorf("failed to remove from index: %w", err)
	}

	// 从Redis删除
	key := r.getInstructionKey(id)
	err = r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete instruction from Redis: %w", err)
	}

	return nil
}

// Helper methods

func (r *RedisSettlementRepository) getInstructionKey(id string) string {
	return r.prefix + "instruction:" + id
}

func (r *RedisSettlementRepository) getRefIndexKey(ref string) string {
	return r.prefix + "index:ref:" + ref
}

func (r *RedisSettlementRepository) getTradeIndexKey(tradeID string) string {
	return r.prefix + "index:trade:" + tradeID
}

func (r *RedisSettlementRepository) getBatchIndexKey(batchID string) string {
	return r.prefix + "index:batch:" + batchID
}

func (r *RedisSettlementRepository) getDateIndexKey(date time.Time) string {
	return r.prefix + "index:date:" + date.Format("2006-01-02")
}

func (r *RedisSettlementRepository) getStatusIndexKey(status domain.SettlementStatus) string {
	return r.prefix + "index:status:" + string(status)
}

func (r *RedisSettlementRepository) addToIndex(ctx context.Context, indexType, id string, date time.Time) error {
	// 添加到日期索引
	dateKey := r.getDateIndexKey(date)
	err := r.client.SAdd(ctx, dateKey, id).Err()
	if err != nil {
		return fmt.Errorf("failed to add to date index: %w", err)
	}

	// 设置过期时间（30天后）
	err = r.client.Expire(ctx, dateKey, 30*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration for date index: %w", err)
	}

	return nil
}

func (r *RedisSettlementRepository) removeFromIndex(ctx context.Context, indexType, id string, date time.Time) error {
	// 从日期索引移除
	dateKey := r.getDateIndexKey(date)
	err := r.client.SRem(ctx, dateKey, id).Err()
	if err != nil {
		return fmt.Errorf("failed to remove from date index: %w", err)
	}

	return nil
}

func (r *RedisSettlementRepository) updateStatusIndex(ctx context.Context, oldInstruction, newInstruction *domain.SettlementInstruction) error {
	// 从旧状态索引移除
	oldStatusKey := r.getStatusIndexKey(oldInstruction.Status)
	err := r.client.SRem(ctx, oldStatusKey, oldInstruction.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove from old status index: %w", err)
	}

	// 添加到新状态索引
	newStatusKey := r.getStatusIndexKey(newInstruction.Status)
	err = r.client.SAdd(ctx, newStatusKey, newInstruction.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to add to new status index: %w", err)
	}

	return nil
}

// RedisSettlementBatchRepository Redis实现的结算批次仓储
type RedisSettlementBatchRepository struct {
	client *redis.Client
	prefix string
}

// NewRedisSettlementBatchRepository 创建Redis结算批次仓储
func NewRedisSettlementBatchRepository(client *redis.Client) *RedisSettlementBatchRepository {
	return &RedisSettlementBatchRepository{
		client: client,
		prefix: "settlement:batch:",
	}
}

// SaveBatch 保存结算批次
func (r *RedisSettlementBatchRepository) SaveBatch(ctx context.Context, batch *domain.SettlementBatch) error {
	key := r.getBatchKey(batch.ID)

	// 序列化批次
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	// 保存到Redis
	err = r.client.Set(ctx, key, data, 30*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save batch to Redis: %w", err)
	}

	// 添加到索引
	err = r.addToIndex(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to add to index: %w", err)
	}

	return nil
}

// GetBatch 获取结算批次
func (r *RedisSettlementBatchRepository) GetBatch(ctx context.Context, id string) (*domain.SettlementBatch, error) {
	key := r.getBatchKey(id)

	// 从Redis获取
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("batch not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get batch from Redis: %w", err)
	}

	// 反序列化
	var batch domain.SettlementBatch
	err = json.Unmarshal(data, &batch)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch: %w", err)
	}

	return &batch, nil
}

// GetBatchByDate 通过日期获取结算批次
func (r *RedisSettlementBatchRepository) GetBatchByDate(ctx context.Context, settlementDate time.Time, cycle domain.SettlementCycle) (*domain.SettlementBatch, error) {
	// 创建日期索引键
	dateKey := r.getDateIndexKey(settlementDate, cycle)

	// 获取批次ID
	batchID, err := r.client.Get(ctx, dateKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 批次不存在
		}
		return nil, fmt.Errorf("failed to get batch ID from Redis: %w", err)
	}

	// 获取批次
	return r.GetBatch(ctx, batchID)
}

// GetBatchesByStatus 通过状态获取结算批次
func (r *RedisSettlementBatchRepository) GetBatchesByStatus(ctx context.Context, status string, startDate, endDate time.Time) ([]*domain.SettlementBatch, error) {
	// 创建状态索引键
	statusKey := r.getStatusIndexKey(status)

	// 获取批次ID列表
	batchIDs, err := r.client.SMembers(ctx, statusKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get batch IDs from Redis: %w", err)
	}

	var batches []*domain.SettlementBatch
	for _, id := range batchIDs {
		batch, err := r.GetBatch(ctx, id)
		if err != nil {
			// 记录错误但继续处理其他批次
			fmt.Printf("Failed to get batch %s: %v\n", id, err)
			continue
		}

		// 过滤日期范围
		if batch.SettlementDate.After(startDate) && batch.SettlementDate.Before(endDate) {
			batches = append(batches, batch)
		}
	}

	return batches, nil
}

// UpdateBatch 更新结算批次
func (r *RedisSettlementBatchRepository) UpdateBatch(ctx context.Context, batch *domain.SettlementBatch) error {
	// 先获取现有批次
	existing, err := r.GetBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing batch: %w", err)
	}

	// 检查状态变化
	if existing.Status != batch.Status {
		// 更新状态索引
		err = r.updateStatusIndex(ctx, existing, batch)
		if err != nil {
			return fmt.Errorf("failed to update status index: %w", err)
		}
	}

	// 保存更新后的批次
	return r.SaveBatch(ctx, batch)
}

// DeleteBatch 删除结算批次
func (r *RedisSettlementBatchRepository) DeleteBatch(ctx context.Context, id string) error {
	// 获取批次
	batch, err := r.GetBatch(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	// 从索引中移除
	err = r.removeFromIndex(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to remove from index: %w", err)
	}

	// 从Redis删除
	key := r.getBatchKey(id)
	err = r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete batch from Redis: %w", err)
	}

	return nil
}

// Helper methods

func (r *RedisSettlementBatchRepository) getBatchKey(id string) string {
	return r.prefix + "data:" + id
}

func (r *RedisSettlementBatchRepository) getDateIndexKey(date time.Time, cycle domain.SettlementCycle) string {
	return r.prefix + "index:date:" + date.Format("2006-01-02") + ":" + string(cycle)
}

func (r *RedisSettlementBatchRepository) getStatusIndexKey(status string) string {
	return r.prefix + "index:status:" + status
}

func (r *RedisSettlementBatchRepository) addToIndex(ctx context.Context, batch *domain.SettlementBatch) error {
	// 添加到日期索引
	dateKey := r.getDateIndexKey(batch.SettlementDate, batch.Cycle)
	err := r.client.Set(ctx, dateKey, batch.ID, 30*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to add to date index: %w", err)
	}

	// 添加到状态索引
	statusKey := r.getStatusIndexKey(batch.Status)
	err = r.client.SAdd(ctx, statusKey, batch.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to add to status index: %w", err)
	}

	return nil
}

func (r *RedisSettlementBatchRepository) removeFromIndex(ctx context.Context, batch *domain.SettlementBatch) error {
	// 从日期索引移除
	dateKey := r.getDateIndexKey(batch.SettlementDate, batch.Cycle)
	err := r.client.Del(ctx, dateKey).Err()
	if err != nil {
		return fmt.Errorf("failed to remove from date index: %w", err)
	}

	// 从状态索引移除
	statusKey := r.getStatusIndexKey(batch.Status)
	err = r.client.SRem(ctx, statusKey, batch.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove from status index: %w", err)
	}

	return nil
}

func (r *RedisSettlementBatchRepository) updateStatusIndex(ctx context.Context, oldBatch, newBatch *domain.SettlementBatch) error {
	// 从旧状态索引移除
	oldStatusKey := r.getStatusIndexKey(oldBatch.Status)
	err := r.client.SRem(ctx, oldStatusKey, oldBatch.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove from old status index: %w", err)
	}

	// 添加到新状态索引
	newStatusKey := r.getStatusIndexKey(newBatch.Status)
	err = r.client.SAdd(ctx, newStatusKey, newBatch.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to add to new status index: %w", err)
	}

	return nil
}
