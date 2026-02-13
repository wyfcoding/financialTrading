package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/messagequeue"
)

// PositionCommandService 处理头寸相关的命令操作
// 使用 Outbox 发布领域事件

type PositionCommandService struct {
	repo           domain.PositionRepository
	eventPublisher messagequeue.EventPublisher
}

// NewPositionCommandService 创建新的 PositionCommandService 实例
func NewPositionCommandService(repo domain.PositionRepository, eventPublisher messagequeue.EventPublisher) *PositionCommandService {
	return &PositionCommandService{
		repo:           repo,
		eventPublisher: eventPublisher,
	}
}

// UpdatePosition 更新头寸
func (c *PositionCommandService) UpdatePosition(ctx context.Context, cmd UpdatePositionCommand) (*domain.Position, error) {
	if cmd.UserID == "" || cmd.Symbol == "" {
		return nil, errors.New("user_id and symbol are required")
	}

	var updated *domain.Position
	err := c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)

		// 获取或创建头寸
		position, err := c.repo.GetByUserSymbol(txCtx, cmd.UserID, cmd.Symbol)
		if err != nil {
			return err
		}
		if position == nil {
			position = domain.NewPosition(cmd.UserID, cmd.Symbol)
			if err := c.repo.Save(txCtx, position); err != nil {
				return err
			}

			if c.eventPublisher != nil {
				createdEvent := domain.PositionCreatedEvent{
					UserID:            position.UserID,
					Symbol:            position.Symbol,
					Quantity:          position.Quantity,
					AverageEntryPrice: position.AverageEntryPrice,
					Method:            position.Method,
					OccurredOn:        time.Now(),
				}
				if err := c.eventPublisher.PublishInTx(txCtx, tx, domain.PositionCreatedEventType, positionKey(position), createdEvent); err != nil {
					return err
				}
			}
		}

		if err := c.applyTrade(txCtx, tx, position, cmd.Side, cmd.Quantity, cmd.Price); err != nil {
			return err
		}

		updated = position
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// ChangeCostMethod 变更成本计算方法
func (c *PositionCommandService) ChangeCostMethod(ctx context.Context, cmd ChangeCostMethodCommand) error {
	if cmd.UserID == "" || cmd.Symbol == "" {
		return errors.New("user_id and symbol are required")
	}

	return c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)

		position, err := c.repo.GetByUserSymbol(txCtx, cmd.UserID, cmd.Symbol)
		if err != nil {
			return err
		}
		if position == nil {
			return errors.New("position not found")
		}

		oldMethod := position.Method
		newMethod := domain.CostBasisMethod(cmd.Method)
		if newMethod == "" {
			newMethod = domain.CostBasisAverage
		}
		if oldMethod == newMethod {
			return nil
		}

		position.Method = newMethod
		if err := c.repo.Save(txCtx, position); err != nil {
			return err
		}

		if c.eventPublisher != nil {
			methodChangedEvent := domain.PositionCostMethodChangedEvent{
				UserID:     position.UserID,
				Symbol:     position.Symbol,
				OldMethod:  oldMethod,
				NewMethod:  newMethod,
				ChangedAt:  time.Now().Unix(),
				OccurredOn: time.Now(),
			}
			return c.eventPublisher.PublishInTx(txCtx, tx, domain.PositionCostMethodChangedEventType, positionKey(position), methodChangedEvent)
		}
		return nil
	})
}

// ClosePosition 平仓
func (c *PositionCommandService) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	if positionID == "" {
		return errors.New("position_id is required")
	}
	return c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)

		position, err := c.repo.Get(txCtx, positionID)
		if err != nil {
			return err
		}
		if position == nil || position.Quantity.IsZero() {
			return nil
		}

		side := "sell"
		if position.Quantity.IsNegative() {
			side = "buy"
		}
		qty := position.Quantity.Abs()
		return c.applyTrade(txCtx, tx, position, side, qty, closePrice)
	})
}

// TccTryFreeze TCC 尝试冻结
func (c *PositionCommandService) TccTryFreeze(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}

	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		position, err := c.repo.GetByUserSymbol(txCtx, userID, symbol)
		if err != nil {
			return err
		}
		if position == nil {
			return errors.New("position not found")
		}
		if position.Quantity.Abs().LessThan(quantity) {
			return errors.New("insufficient position quantity to freeze")
		}
		return c.publishTxEvent(txCtx, domain.PositionTccTryFrozenEventType, fmt.Sprintf("%s:%s:%s", userID, symbol, quantity.String()), map[string]any{
			"user_id":  userID,
			"symbol":   symbol,
			"quantity": quantity.String(),
		})
	})
}

// TccConfirmFreeze TCC 确认冻结
func (c *PositionCommandService) TccConfirmFreeze(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}

	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)
		position, err := c.repo.GetByUserSymbol(txCtx, userID, symbol)
		if err != nil {
			return err
		}
		if position == nil {
			return errors.New("position not found")
		}
		if position.Quantity.Abs().LessThan(quantity) {
			return errors.New("insufficient position quantity for confirm")
		}

		if err := c.applyTrade(txCtx, tx, position, sideForReduce(position), quantity, fallbackPrice(position)); err != nil {
			return err
		}
		return c.publishTxEvent(txCtx, domain.PositionTccConfirmedEventType, fmt.Sprintf("%s:%s:%s", userID, symbol, quantity.String()), map[string]any{
			"user_id":  userID,
			"symbol":   symbol,
			"quantity": quantity.String(),
		})
	})
}

// TccCancelFreeze TCC 取消冻结
func (c *PositionCommandService) TccCancelFreeze(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}
	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		return c.publishTxEvent(txCtx, domain.PositionTccCanceledEventType, fmt.Sprintf("%s:%s:%s", userID, symbol, quantity.String()), map[string]any{
			"user_id":  userID,
			"symbol":   symbol,
			"quantity": quantity.String(),
		})
	})
}

// SagaDeductFrozen SAGA 扣减冻结
func (c *PositionCommandService) SagaDeductFrozen(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}

	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)
		position, err := c.repo.GetByUserSymbol(txCtx, userID, symbol)
		if err != nil {
			return err
		}
		if position == nil {
			return errors.New("position not found")
		}
		if position.Quantity.Abs().LessThan(quantity) {
			return errors.New("insufficient position quantity to deduct")
		}
		if err := c.applyTrade(txCtx, tx, position, sideForReduce(position), quantity, price); err != nil {
			return err
		}
		return c.publishTxEvent(txCtx, domain.PositionSagaDeductedEventType, fmt.Sprintf("%s:%s:%s", userID, symbol, quantity.String()), map[string]any{
			"user_id":  userID,
			"symbol":   symbol,
			"quantity": quantity.String(),
			"price":    price.String(),
		})
	})
}

// SagaRefundFrozen SAGA 退还冻结
func (c *PositionCommandService) SagaRefundFrozen(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}

	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)
		position, err := c.ensurePosition(txCtx, tx, userID, symbol)
		if err != nil {
			return err
		}
		if err := c.applyTrade(txCtx, tx, position, sideForIncrease(position), quantity, fallbackPrice(position)); err != nil {
			return err
		}
		return c.publishTxEvent(txCtx, domain.PositionSagaRefundedEventType, fmt.Sprintf("%s:%s:%s", userID, symbol, quantity.String()), map[string]any{
			"user_id":  userID,
			"symbol":   symbol,
			"quantity": quantity.String(),
		})
	})
}

// SagaAddPosition SAGA 增加头寸
func (c *PositionCommandService) SagaAddPosition(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}

	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)
		position, err := c.ensurePosition(txCtx, tx, userID, symbol)
		if err != nil {
			return err
		}
		return c.applyTrade(txCtx, tx, position, "buy", quantity, price)
	})
}

// SagaSubPosition SAGA 减少头寸
func (c *PositionCommandService) SagaSubPosition(ctx context.Context, barrier any, userID string, symbol string, quantity decimal.Decimal) error {
	if userID == "" || symbol == "" {
		return errors.New("user_id and symbol are required")
	}
	if quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("quantity must be positive")
	}

	return c.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)
		position, err := c.repo.GetByUserSymbol(txCtx, userID, symbol)
		if err != nil {
			return err
		}
		if position == nil {
			return errors.New("position not found")
		}
		if position.Quantity.Abs().LessThan(quantity) {
			return errors.New("insufficient position quantity")
		}
		return c.applyTrade(txCtx, tx, position, "sell", quantity, fallbackPrice(position))
	})
}

func (c *PositionCommandService) ensurePosition(ctx context.Context, tx any, userID, symbol string) (*domain.Position, error) {
	position, err := c.repo.GetByUserSymbol(ctx, userID, symbol)
	if err != nil {
		return nil, err
	}
	if position != nil {
		return position, nil
	}

	position = domain.NewPosition(userID, symbol)
	if err := c.repo.Save(ctx, position); err != nil {
		return nil, err
	}
	if c.eventPublisher != nil {
		createdEvent := domain.PositionCreatedEvent{
			UserID:            position.UserID,
			Symbol:            position.Symbol,
			Quantity:          position.Quantity,
			AverageEntryPrice: position.AverageEntryPrice,
			Method:            position.Method,
			OccurredOn:        time.Now(),
		}
		if err := c.eventPublisher.PublishInTx(ctx, tx, domain.PositionCreatedEventType, positionKey(position), createdEvent); err != nil {
			return nil, err
		}
	}
	return position, nil
}

func (c *PositionCommandService) publishTxEvent(ctx context.Context, topic, key string, payload map[string]any) error {
	if c.eventPublisher == nil {
		return nil
	}
	tx := contextx.GetTx(ctx)
	return c.eventPublisher.PublishInTx(ctx, tx, topic, key, payload)
}

func sideForReduce(position *domain.Position) string {
	if position != nil && position.Quantity.IsNegative() {
		return "buy"
	}
	return "sell"
}

func sideForIncrease(position *domain.Position) string {
	if position != nil && position.Quantity.IsNegative() {
		return "sell"
	}
	return "buy"
}

func fallbackPrice(position *domain.Position) decimal.Decimal {
	if position == nil || position.AverageEntryPrice.IsZero() {
		return decimal.NewFromInt(1)
	}
	return position.AverageEntryPrice
}

func (c *PositionCommandService) applyTrade(ctx context.Context, tx any, position *domain.Position, side string, qty, price decimal.Decimal) error {
	oldQuantity := position.Quantity
	oldAveragePrice := position.AverageEntryPrice
	oldRealizedPnL := position.RealizedPnL

	position.UpdatePosition(side, qty, price)

	if err := c.repo.Save(ctx, position); err != nil {
		return err
	}

	if c.eventPublisher == nil {
		return nil
	}

	updatedEvent := domain.PositionUpdatedEvent{
		UserID:          position.UserID,
		Symbol:          position.Symbol,
		OldQuantity:     oldQuantity,
		NewQuantity:     position.Quantity,
		OldAveragePrice: oldAveragePrice,
		NewAveragePrice: position.AverageEntryPrice,
		TradeSide:       side,
		TradeQuantity:   qty,
		TradePrice:      price,
		OccurredOn:      time.Now(),
	}
	if err := c.eventPublisher.PublishInTx(ctx, tx, domain.PositionUpdatedEventType, positionKey(position), updatedEvent); err != nil {
		return err
	}

	pnlChange := position.RealizedPnL.Sub(oldRealizedPnL)
	if !pnlChange.IsZero() {
		pnlEvent := domain.PositionPnLUpdatedEvent{
			UserID:         position.UserID,
			Symbol:         position.Symbol,
			OldRealizedPnL: oldRealizedPnL,
			NewRealizedPnL: position.RealizedPnL,
			TradeQuantity:  qty,
			TradePrice:     price,
			PnLChange:      pnlChange,
			UpdatedAt:      time.Now().Unix(),
			OccurredOn:     time.Now(),
		}
		if err := c.eventPublisher.PublishInTx(ctx, tx, domain.PositionPnLUpdatedEventType, positionKey(position), pnlEvent); err != nil {
			return err
		}
	}

	if position.Quantity.IsZero() && !oldQuantity.IsZero() {
		closedEvent := domain.PositionClosedEvent{
			UserID:        position.UserID,
			Symbol:        position.Symbol,
			FinalQuantity: position.Quantity,
			RealizedPnL:   position.RealizedPnL,
			ClosedAt:      time.Now().Unix(),
			OccurredOn:    time.Now(),
		}
		if err := c.eventPublisher.PublishInTx(ctx, tx, domain.PositionClosedEventType, positionKey(position), closedEvent); err != nil {
			return err
		}
	}

	if (oldQuantity.IsPositive() && position.Quantity.IsNegative()) || (oldQuantity.IsNegative() && position.Quantity.IsPositive()) {
		oldDirection := "short"
		if oldQuantity.IsPositive() {
			oldDirection = "long"
		}

		newDirection := "short"
		if position.Quantity.IsPositive() {
			newDirection = "long"
		}

		flipEvent := domain.PositionFlipEvent{
			UserID:       position.UserID,
			Symbol:       position.Symbol,
			OldDirection: oldDirection,
			NewDirection: newDirection,
			FlipQuantity: qty,
			FlipPrice:    price,
			OccurredOn:   time.Now(),
		}
		if err := c.eventPublisher.PublishInTx(ctx, tx, domain.PositionFlipEventType, positionKey(position), flipEvent); err != nil {
			return err
		}
	}

	return nil
}

func positionKey(position *domain.Position) string {
	if position == nil {
		return ""
	}
	if position.ID != 0 {
		return fmt.Sprintf("%d", position.ID)
	}
	if position.UserID != "" || position.Symbol != "" {
		return fmt.Sprintf("%s:%s", position.UserID, position.Symbol)
	}
	return ""
}
