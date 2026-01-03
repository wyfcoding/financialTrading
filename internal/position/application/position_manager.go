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

// UpdatePositionPrice 更新持仓当前价格
func (m *PositionManager) UpdatePositionPrice(ctx context.Context, positionID string, currentPrice decimal.Decimal) error {
	pos, err := m.repo.Get(ctx, positionID)
	if err != nil {
		return err
	}
	if pos == nil {
		return fmt.Errorf("position not found")
	}

	pos.CurrentPrice = currentPrice
	return m.repo.Update(ctx, pos)
}

// --- TCC Distributed Transaction Support ---

// TccTryFreeze TCC Try: 预冻结持仓资产
func (m *PositionManager) TccTryFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 1. 查找用户在该币种的持仓 (此处简化：假设 symbol 就是持仓标的)
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

		// 2. 检查可用持仓数量
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
}

// TccConfirmFreeze TCC Confirm: 确认冻结 (Try 阶段已扣减，此处为空操作)
func (m *PositionManager) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return nil
	})
}

// TccCancelFreeze TCC Cancel: 取消冻结 (恢复持仓)
func (m *PositionManager) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
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

		if targetPos != nil {
			targetPos.Quantity = targetPos.Quantity.Add(quantity)
			return m.repo.Update(ctx, targetPos)
		}
		return nil
	})
}

// --- Saga Distributed Transaction Support ---

// SagaDeductFrozen Saga 正向: 扣除已冻结持仓 (成交确认，盈亏结转)
func (m *PositionManager) SagaDeductFrozen(ctx context.Context, barrier interface{}, userID, symbol string, quantity, price decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 1. 查找用户在该币种的持仓
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
			// 在 TCC 模式下，卖单必定已有持仓被冻结，此处若为空说明数据异常
			return fmt.Errorf("position not found for PnL realization: user=%s, symbol=%s", userID, symbol)
		}

		// 2. 执行盈亏核算
		// 注意：此时数量在 TCC Try 阶段已经从 Quantity 中扣除，此处主要结转盈亏
		targetPos.RealizePnL(quantity, price)

		// 3. 更新持仓记录
		return m.repo.Update(ctx, targetPos)
	})
}

// SagaRefundFrozen Saga 补偿: 恢复冻结持仓
func (m *PositionManager) SagaRefundFrozen(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		// 恢复扣除的资产
		return m.TccCancelFreeze(ctx, barrier, userID, symbol, quantity)
	})
}

// SagaAddPosition Saga 正向: 增加持仓 (买方资产入账，带开仓均价计算)
func (m *PositionManager) SagaAddPosition(ctx context.Context, barrier interface{}, userID, symbol string, quantity, price decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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
			// 创建新持仓
			targetPos = &domain.Position{
				PositionID:   fmt.Sprintf("POS-%d", idgen.GenID()),
				UserID:       userID,
				Symbol:       symbol,
				Side:         "LONG", // 默认多头
				Quantity:     quantity,
				EntryPrice:   price, // 第一笔成交价即为初始均价
				CurrentPrice: price,
				Status:       "OPEN",
				OpenedAt:     time.Now(),
			}
			return m.repo.Save(ctx, targetPos)
		}

		// 使用领域层逻辑滚动计算均价
		targetPos.AddQuantity(quantity, price)
		return m.repo.Update(ctx, targetPos)
	})
}

// SagaSubPosition Saga 补偿: 扣除已增加的持仓
func (m *PositionManager) SagaSubPosition(ctx context.Context, barrier interface{}, userID, symbol string, quantity decimal.Decimal) error {
	return m.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
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
}
