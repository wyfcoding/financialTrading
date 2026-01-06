package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/idgen"
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

// UpdatePositionPrice 实时更新持仓的标记价格，用于动态计算未实现盈亏。
func (m *PositionManager) UpdatePositionPrice(ctx context.Context, positionID string, currentPrice decimal.Decimal) error {
	pos, err := m.repo.Get(ctx, positionID)
	if err != nil {
		return err
	}
	if pos == nil {
		return fmt.Errorf("position not found")
	}

	pos.CurrentPrice = currentPrice
	if err := m.repo.Update(ctx, pos); err != nil {
		return err
	}

	logging.Debug(ctx, "position price updated", "position_id", positionID, "new_price", currentPrice.String())
	return nil
}

// --- TCC Distributed Transaction Support ---

// TccTryFreeze 执行 TCC 第一阶段：预冻结持仓资产（减少可用持仓）。
func (m *PositionManager) TccTryFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	err := m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		var targetPos *domain.Position
		for _, p := range positions {
			if p.Symbol == symbol && p.Status == "OPEN" {
				targetPos = p
				break
			}
		}

		if targetPos == nil {
			return fmt.Errorf("no open position found for %s", symbol)
		}

		if targetPos.Quantity.LessThan(quantity) {
			return fmt.Errorf("insufficient position quantity to sell: have %s, need %s", targetPos.Quantity, quantity)
		}

		// 3. 执行冻结 (减少总持仓，直到结算再处理，或者增加一个 FrozenQuantity 字段)
		// 为了简单起见且不修改模型，我们在此处直接扣减 Quantity。
		// 实际上，生产环境通常有专门的 FrozenQuantity 字段。
		// 此处演示：减少 Quantity 并更新状态。
		targetPos.Quantity = targetPos.Quantity.Sub(quantity)
		return m.repo.Update(ctx, targetPos)
	})

	if err != nil {
		logging.Error(ctx, "tcc_try_freeze failed", "user_id", userID, "symbol", symbol, "error", err)
		return err
	}
	logging.Debug(ctx, "tcc_try_freeze successful", "user_id", userID, "symbol", symbol, "qty", quantity.String())
	return nil
}

// TccConfirmFreeze 执行 TCC 第二阶段：确认冻结。
func (m *PositionManager) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return nil
	})
}

// TccCancelFreeze 执行 TCC 取消阶段：恢复之前预冻结的持仓资产。
func (m *PositionManager) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	err := m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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

		if targetPos != nil {
			targetPos.Quantity = targetPos.Quantity.Add(quantity)
			return m.repo.Update(ctx, targetPos)
		}
		return nil
	})

	if err != nil {
		logging.Error(ctx, "tcc_cancel_freeze failed", "user_id", userID, "symbol", symbol, "error", err)
		return err
	}
	logging.Info(ctx, "tcc_cancel_freeze successful", "user_id", userID, "symbol", symbol, "qty", quantity.String())
	return nil
}

// --- Saga Distributed Transaction Support ---

// SagaDeductFrozen 执行 Saga 正向流程：扣除已冻结持仓并结转盈亏。
func (m *PositionManager) SagaDeductFrozen(ctx context.Context, barrier interface{}, userID, symbol string, quantity, price decimal.Decimal) error {
	err := m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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
			return fmt.Errorf("position not found for pnl realization: user=%s, symbol=%s", userID, symbol)
		}

		targetPos.RealizePnL(quantity, price)
		return m.repo.Update(ctx, targetPos)
	})

	if err != nil {
		logging.Error(ctx, "saga_deduct_frozen failed", "user_id", userID, "symbol", symbol, "error", err)
		return err
	}
	logging.Info(ctx, "saga_deduct_frozen successful", "user_id", userID, "symbol", symbol, "qty", quantity.String())
	return nil
}

// SagaRefundFrozen 执行 Saga 补偿流程：恢复已扣除的冻结持仓。
func (m *PositionManager) SagaRefundFrozen(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	err := m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return m.TccCancelFreeze(ctx, barrier, userID, symbol, quantity)
	})

	if err != nil {
		logging.Error(ctx, "saga_refund_frozen failed", "user_id", userID, "symbol", symbol, "error", err)
		return err
	}
	logging.Info(ctx, "saga_refund_frozen successful", "user_id", userID, "symbol", symbol, "qty", quantity.String())
	return nil
}

// SagaAddPosition 执行 Saga 正向流程：买入资产成功，入账并重新计算成本均价。
func (m *PositionManager) SagaAddPosition(ctx context.Context, barrier interface{}, userID, symbol string, quantity, price decimal.Decimal) error {
	err := m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		var targetPos *domain.Position
		for _, p := range positions {
			if p.Symbol == symbol && p.Status == "OPEN" {
				targetPos = p
				break
			}
		}

		if targetPos == nil {
			targetPos = &domain.Position{
				PositionID:   fmt.Sprintf("POS-%d", idgen.GenID()),
				UserID:       userID,
				Symbol:       symbol,
				Side:         "LONG",
				Quantity:     quantity,
				EntryPrice:   price,
				CurrentPrice: price,
				Status:       "OPEN",
				OpenedAt:     time.Now(),
			}
			return m.repo.Save(ctx, targetPos)
		}

		targetPos.AddQuantity(quantity, price)
		return m.repo.Update(ctx, targetPos)
	})

	if err != nil {
		logging.Error(ctx, "saga_add_position failed", "user_id", userID, "symbol", symbol, "error", err)
		return err
	}
	logging.Info(ctx, "saga_add_position successful", "user_id", userID, "symbol", symbol, "qty", quantity.String())
	return nil
}

// SagaSubPosition 执行 Saga 补偿流程：扣除已增加的持仓。
func (m *PositionManager) SagaSubPosition(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	err := m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		positions, _, err := m.repo.GetByUser(ctx, userID, 100, 0)
		if err != nil {
			return err
		}

		for _, p := range positions {
			if p.Symbol == symbol {
				if p.Quantity.LessThan(quantity) {
					return fmt.Errorf("insufficient quantity to roll back position")
				}
				p.Quantity = p.Quantity.Sub(quantity)
				return m.repo.Update(ctx, p)
			}
		}
		return nil
	})

	if err != nil {
		logging.Error(ctx, "saga_sub_position failed", "user_id", userID, "symbol", symbol, "error", err)
		return err
	}
	logging.Info(ctx, "saga_sub_position successful", "user_id", userID, "symbol", symbol, "qty", quantity.String())
	return nil
}
