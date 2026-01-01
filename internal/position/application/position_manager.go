package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/logging"
)

// PositionManager 处理所有持仓相关的写入操作（Commands）。
type PositionManager struct {
	repo domain.PositionRepository
}

// NewPositionManager 构造函数。
func NewPositionManager(repo domain.PositionRepository) *PositionManager {
	return &PositionManager{repo: repo}
}

// ClosePosition 平仓
func (m *PositionManager) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	if positionID == "" || closePrice.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("invalid request parameters")
	}

	if err := m.repo.Close(ctx, positionID, closePrice); err != nil {
		logging.Error(ctx, "Failed to close position",
			"position_id", positionID,
			"error", err,
		)
		return err
	}

	return nil
}

// UpdatePositionPrice 更新持仓当前价格（领域逻辑通常在此触发盈亏重算）
func (m *PositionManager) UpdatePositionPrice(ctx context.Context, positionID string, currentPrice decimal.Decimal) error {
	pos, err := m.repo.Get(ctx, positionID)
	if err != nil {
		return err
	}
	if pos == nil {
		return fmt.Errorf("position not found")
	}

	pos.CurrentPrice = currentPrice
	// 在此可以调用领域层 PNL 计算 logic
	// pos.UnrealizedPnL = ...

	return m.repo.Update(ctx, pos)
}
