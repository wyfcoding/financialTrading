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

// ClosePosition 彻底平仓指定的持仓。
func (m *PositionManager) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	if positionID == "" || closePrice.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("invalid request parameters")
	}

	if err := m.repo.Close(ctx, positionID, closePrice); err != nil {
		logging.Error(ctx, "failed to close position", "position_id", positionID, "error", err)
		return err
	}

	logging.Info(ctx, "position closed successfully", "position_id", positionID, "close_price", closePrice.String())
	return nil
}

// --- TCC Distributed Transaction Support ---

// TccTryFreeze 执行 TCC 第一阶段：预冻结持仓资产（减少可用持仓）。
func (m *PositionManager) TccTryFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		var targetPos *domain.Position
		for _, p := range positions {
			if p.Symbol == symbol {
				targetPos = p
				break
			}
		}

		if targetPos == nil {
			return fmt.Errorf("no open position found for %s", symbol)
		}

		requestQty := quantity.InexactFloat64()
		if targetPos.Quantity < requestQty {
			return fmt.Errorf("insufficient position quantity to sell: have %f, need %f", targetPos.Quantity, requestQty)
		}

		// 3. 执行冻结 (减少总持仓)
		targetPos.Quantity -= requestQty
		return m.repo.Update(ctx, targetPos)
	})
}

// TccConfirmFreeze 执行 TCC 第二阶段：确认冻结。
func (m *PositionManager) TccConfirmFreeze(ctx context.Context, barrier any, _, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return nil
	})
}

// TccCancelFreeze 执行 TCC 取消阶段：恢复之前预冻结的持仓资产。
func (m *PositionManager) TccCancelFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		var targetPos *domain.Position
		for _, p := range positions {
			if p.Symbol == symbol {
				targetPos = p
				break
			}
		}

		// If position exists, restore quantity. If closed/gone, might need to re-create?
		// For simplicity, assume existence (it was there in TryFreeze).
		if targetPos != nil {
			targetPos.Quantity += quantity.InexactFloat64()
			return m.repo.Update(ctx, targetPos)
		}

		// If position gone, recreate it?
		// Log warning.
		return fmt.Errorf("position not found to rollback freeze")
	})
}

// --- Saga Distributed Transaction Support ---

// SagaDeductFrozen 执行 Saga 正向流程：扣除已冻结持仓并结转盈亏。
func (m *PositionManager) SagaDeductFrozen(ctx context.Context, barrier any, userID, symbol string, quantity, price decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		var targetPos *domain.Position
		for _, p := range positions {
			if p.Symbol == symbol {
				targetPos = p
				break
			}
		}

		if targetPos == nil {
			return fmt.Errorf("position not found for pnl realization")
		}

		targetPos.UpdatePosition("sell", quantity.InexactFloat64(), price.InexactFloat64())
		return m.repo.Update(ctx, targetPos)
	})
}

// SagaRefundFrozen 执行 Saga 补偿流程：恢复已扣除的冻结持仓。
func (m *PositionManager) SagaRefundFrozen(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return m.TccCancelFreeze(ctx, barrier, userID, symbol, quantity)
}

// SagaAddPosition 执行 Saga 正向流程：买入资产成功，入账并重新计算成本均价。
func (m *PositionManager) SagaAddPosition(ctx context.Context, barrier any, userID, symbol string, quantity, price decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// Use UpdatePositionWithLock logic if available or just get/update
		// Here we reuse get logic
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		var targetPos *domain.Position
		for _, p := range positions {
			if p.Symbol == symbol {
				targetPos = p
				break
			}
		}

		if targetPos == nil {
			targetPos = domain.NewPosition(userID, symbol)
			targetPos.UpdatePosition("buy", quantity.InexactFloat64(), price.InexactFloat64())
			return m.repo.Save(ctx, targetPos)
		}

		targetPos.UpdatePosition("buy", quantity.InexactFloat64(), price.InexactFloat64())
		return m.repo.Update(ctx, targetPos)
	})
}

// SagaSubPosition 执行 Saga 补偿流程：扣除已增加的持仓。
func (m *PositionManager) SagaSubPosition(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		for _, p := range positions {
			if p.Symbol == symbol {
				// Rollback buy -> Sell
				// Use price 0 or original? To rollback perfectly requires original price.
				// For approximation, just reduce quantity?
				// targetPos.Quantity -= qty.
				// This might affect avg price logic if valid.
				// If simple rollback:
				p.Quantity -= quantity.InexactFloat64()
				return m.repo.Update(ctx, p)
			}
		}
		return nil
	})
}
