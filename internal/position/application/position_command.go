package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/contextx"
)

// PositionCommandService 处理头寸相关的命令操作
// 使用 Outbox 发布领域事件

type PositionCommandService struct {
	repo           domain.PositionRepository
	eventPublisher domain.EventPublisher
}

// NewPositionCommandService 创建新的 PositionCommandService 实例
func NewPositionCommandService(repo domain.PositionRepository, eventPublisher domain.EventPublisher) *PositionCommandService {
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
		if position == nil || position.Quantity == 0 {
			return nil
		}

		side := "sell"
		if position.Quantity < 0 {
			side = "buy"
		}
		qty := math.Abs(position.Quantity)
		return c.applyTrade(txCtx, tx, position, side, qty, closePrice.InexactFloat64())
	})
}

// TccTryFreeze TCC 尝试冻结
func (c *PositionCommandService) TccTryFreeze(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	return nil
}

// TccConfirmFreeze TCC 确认冻结
func (c *PositionCommandService) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	return nil
}

// TccCancelFreeze TCC 取消冻结
func (c *PositionCommandService) TccCancelFreeze(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	return nil
}

// SagaDeductFrozen SAGA 扣减冻结
func (c *PositionCommandService) SagaDeductFrozen(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	return nil
}

// SagaRefundFrozen SAGA 退还冻结
func (c *PositionCommandService) SagaRefundFrozen(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	return nil
}

// SagaAddPosition SAGA 增加头寸
func (c *PositionCommandService) SagaAddPosition(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	return nil
}

// SagaSubPosition SAGA 减少头寸
func (c *PositionCommandService) SagaSubPosition(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	return nil
}

func (c *PositionCommandService) applyTrade(ctx context.Context, tx any, position *domain.Position, side string, qty, price float64) error {
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

	pnlChange := position.RealizedPnL - oldRealizedPnL
	if math.Abs(pnlChange) > 0 {
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

	if position.Quantity == 0 && oldQuantity != 0 {
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

	if (oldQuantity > 0 && position.Quantity < 0) || (oldQuantity < 0 && position.Quantity > 0) {
		oldDirection := "short"
		if oldQuantity > 0 {
			oldDirection = "long"
		}

		newDirection := "short"
		if position.Quantity > 0 {
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
