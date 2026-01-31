package application

import (
	"context"
	"math"
	"time"

	"github.com/wyfcoding/financialtrading/internal/position/domain"
)

// UpdatePositionCommand 更新头寸命令
type UpdatePositionCommand struct {
	UserID   string
	Symbol   string
	Side     string
	Quantity float64
	Price    float64
}

// ChangeCostMethodCommand 变更成本计算方法命令
type ChangeCostMethodCommand struct {
	UserID string
	Symbol string
	Method string
}

// PositionCommand 处理头寸相关的命令操作
type PositionCommand struct {
	repo           domain.PositionRepository
	eventPublisher domain.EventPublisher
}

// NewPositionCommand 创建新的 PositionCommand 实例
func NewPositionCommand(repo domain.PositionRepository, eventPublisher domain.EventPublisher) *PositionCommand {
	return &PositionCommand{
		repo:           repo,
		eventPublisher: eventPublisher,
	}
}

// UpdatePosition 更新头寸
func (c *PositionCommand) UpdatePosition(ctx context.Context, cmd UpdatePositionCommand) (*domain.Position, error) {
	// 获取或创建头寸
	position, err := c.repo.GetByUserSymbol(ctx, cmd.UserID, cmd.Symbol)
	if err != nil {
		// 创建新头寸
		position = domain.NewPosition(cmd.UserID, cmd.Symbol)
		position.Method = domain.CostBasisAverage

		// 保存新头寸
		if err := c.repo.Save(ctx, position); err != nil {
			return nil, err
		}

		// 发布头寸创建事件
		createdEvent := domain.PositionCreatedEvent{
			UserID:            position.UserID,
			Symbol:            position.Symbol,
			Quantity:          position.Quantity,
			AverageEntryPrice: position.AverageEntryPrice,
			Method:            position.Method,
			OccurredOn:        time.Now(),
		}

		c.eventPublisher.PublishPositionCreated(createdEvent)
	}

	// 记录旧值
	oldQuantity := position.Quantity
	oldAveragePrice := position.AverageEntryPrice
	oldRealizedPnL := position.RealizedPnL

	// 更新头寸
	_, _ = position.UpdatePosition(cmd.Side, cmd.Quantity, cmd.Price)

	// 保存头寸
	if err := c.repo.Save(ctx, position); err != nil {
		return nil, err
	}

	// 注意：这里暂时移除对SaveLot和DeleteLot的调用，因为Repository接口中没有定义这些方法
	// 实际应用中需要根据Repository的具体实现来决定如何处理lots

	// 计算 PnL 变化
	pnlChange := position.RealizedPnL - oldRealizedPnL

	// 发布头寸更新事件
	updatedEvent := domain.PositionUpdatedEvent{
		UserID:          position.UserID,
		Symbol:          position.Symbol,
		OldQuantity:     oldQuantity,
		NewQuantity:     position.Quantity,
		OldAveragePrice: oldAveragePrice,
		NewAveragePrice: position.AverageEntryPrice,
		TradeSide:       cmd.Side,
		TradeQuantity:   cmd.Quantity,
		TradePrice:      cmd.Price,
		OccurredOn:      time.Now(),
	}

	c.eventPublisher.PublishPositionUpdated(updatedEvent)

	// 如果 PnL 发生变化，发布盈亏更新事件
	if math.Abs(pnlChange) > 0 {
		pnlEvent := domain.PositionPnLUpdatedEvent{
			UserID:         position.UserID,
			Symbol:         position.Symbol,
			OldRealizedPnL: oldRealizedPnL,
			NewRealizedPnL: position.RealizedPnL,
			TradeQuantity:  cmd.Quantity,
			TradePrice:     cmd.Price,
			PnLChange:      pnlChange,
			UpdatedAt:      time.Now().Unix(),
			OccurredOn:     time.Now(),
		}

		c.eventPublisher.PublishPositionPnLUpdated(pnlEvent)
	}

	// 如果头寸被关闭，发布关闭事件
	if position.Quantity == 0 {
		closedEvent := domain.PositionClosedEvent{
			UserID:        position.UserID,
			Symbol:        position.Symbol,
			FinalQuantity: position.Quantity,
			RealizedPnL:   position.RealizedPnL,
			ClosedAt:      time.Now().Unix(),
			OccurredOn:    time.Now(),
		}

		c.eventPublisher.PublishPositionClosed(closedEvent)
	}

	// 如果头寸方向发生变化，发布反手事件
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
			FlipQuantity: cmd.Quantity,
			FlipPrice:    cmd.Price,
			OccurredOn:   time.Now(),
		}

		c.eventPublisher.PublishPositionFlip(flipEvent)
	}

	return position, nil
}

// ChangeCostMethod 变更成本计算方法
func (c *PositionCommand) ChangeCostMethod(ctx context.Context, cmd ChangeCostMethodCommand) error {
	// 获取头寸
	position, err := c.repo.GetByUserSymbol(ctx, cmd.UserID, cmd.Symbol)
	if err != nil {
		return err
	}

	// 记录旧方法
	oldMethod := position.Method
	newMethod := domain.CostBasisMethod(cmd.Method)

	// 更新方法
	position.Method = newMethod

	// 保存头寸
	if err := c.repo.Save(ctx, position); err != nil {
		return err
	}

	// 发布成本计算方法变更事件
	methodChangedEvent := domain.PositionCostMethodChangedEvent{
		UserID:     position.UserID,
		Symbol:     position.Symbol,
		OldMethod:  oldMethod,
		NewMethod:  newMethod,
		ChangedAt:  time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishPositionCostMethodChanged(methodChangedEvent)
}
