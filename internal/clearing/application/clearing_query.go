package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

// ClearingQuery 处理所有清算相关的查询操作（Queries）。
type ClearingQuery struct {
	settlementRepo domain.SettlementRepository
	eodRepo        domain.EODClearingRepository
}

// NewClearingQuery 构造函数。
func NewClearingQuery(settlementRepo domain.SettlementRepository, eodRepo domain.EODClearingRepository) *ClearingQuery {
	return &ClearingQuery{
		settlementRepo: settlementRepo,
		eodRepo:        eodRepo,
	}
}

// GetClearingStatus 获取日终清算任务状态
func (q *ClearingQuery) GetClearingStatus(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	clearing, err := q.eodRepo.Get(ctx, clearingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get clearing status: %w", err)
	}
	return clearing, nil
}

// GetSettlementHistory 获取结算历史
func (q *ClearingQuery) GetSettlementHistory(ctx context.Context, userID string, limit, offset int) ([]*domain.Settlement, int64, error) {
	return q.settlementRepo.GetByUser(ctx, userID, limit, offset)
}
