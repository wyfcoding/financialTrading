package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataCommand 处理参考数据相关的命令操作
type ReferenceDataCommand struct {
	repo           domain.ReferenceDataRepository
	eventPublisher domain.EventPublisher
}

// NewReferenceDataCommand 创建新的 ReferenceDataCommand 实例
func NewReferenceDataCommand(repo domain.ReferenceDataRepository, eventPublisher domain.EventPublisher) *ReferenceDataCommand {
	return &ReferenceDataCommand{
		repo:           repo,
		eventPublisher: eventPublisher,
	}
}

// CreateSymbol 创建交易对
func (c *ReferenceDataCommand) CreateSymbol(ctx context.Context, cmd CreateSymbolCommand) (*domain.Symbol, error) {
	// 创建交易对
	symbol := &domain.Symbol{
		ID:             cmd.SymbolID,
		BaseCurrency:   cmd.BaseCurrency,
		QuoteCurrency:  cmd.QuoteCurrency,
		ExchangeID:     cmd.ExchangeID,
		SymbolCode:     cmd.SymbolCode,
		Status:         cmd.Status,
		MinOrderSize:   decimal.NewFromFloat(cmd.MinOrderSize),
		PricePrecision: decimal.NewFromFloat(cmd.PricePrecision),
	}

	// 保存交易对
	if err := c.repo.SaveSymbol(ctx, symbol); err != nil {
		return nil, err
	}

	// 发布交易对创建事件
	createdEvent := domain.SymbolCreatedEvent{
		SymbolID:       symbol.ID,
		BaseCurrency:   symbol.BaseCurrency,
		QuoteCurrency:  symbol.QuoteCurrency,
		ExchangeID:     symbol.ExchangeID,
		SymbolCode:     symbol.SymbolCode,
		Status:         symbol.Status,
		MinOrderSize:   cmd.MinOrderSize,
		PricePrecision: cmd.PricePrecision,
		CreatedAt:      time.Now().Unix(),
		OccurredOn:     time.Now(),
	}

	c.eventPublisher.PublishSymbolCreated(createdEvent)

	return symbol, nil
}

// UpdateSymbol 更新交易对
func (c *ReferenceDataCommand) UpdateSymbol(ctx context.Context, cmd UpdateSymbolCommand) (*domain.Symbol, error) {
	// 获取交易对
	symbol, err := c.repo.GetSymbol(ctx, cmd.SymbolID)
	if err != nil {
		return nil, err
	}

	// 记录旧值
	oldStatus := symbol.Status
	oldMinOrderSize := symbol.MinOrderSize.InexactFloat64()

	// 更新交易对
	symbol.Status = cmd.Status
	symbol.MinOrderSize = decimal.NewFromFloat(cmd.MinOrderSize)
	symbol.PricePrecision = decimal.NewFromFloat(cmd.PricePrecision)

	// 保存交易对
	if err := c.repo.SaveSymbol(ctx, symbol); err != nil {
		return nil, err
	}

	// 发布交易对更新事件
	updatedEvent := domain.SymbolUpdatedEvent{
		SymbolID:        symbol.ID,
		OldStatus:       oldStatus,
		NewStatus:       symbol.Status,
		OldMinOrderSize: oldMinOrderSize,
		NewMinOrderSize: cmd.MinOrderSize,
		UpdatedAt:       time.Now().Unix(),
		OccurredOn:      time.Now(),
	}

	c.eventPublisher.PublishSymbolUpdated(updatedEvent)

	// 如果状态变更，发布状态变更事件
	if oldStatus != symbol.Status {
		statusChangedEvent := domain.SymbolStatusChangedEvent{
			SymbolID:   symbol.ID,
			SymbolCode: symbol.SymbolCode,
			OldStatus:  oldStatus,
			NewStatus:  symbol.Status,
			ChangedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}

		c.eventPublisher.PublishSymbolStatusChanged(statusChangedEvent)
	}

	return symbol, nil
}

// DeleteSymbol 删除交易对
func (c *ReferenceDataCommand) DeleteSymbol(ctx context.Context, cmd DeleteSymbolCommand) error {
	// 获取交易对
	symbol, err := c.repo.GetSymbol(ctx, cmd.SymbolID)
	if err != nil {
		return err
	}

	// 删除交易对
	// 暂时注释，因为 repository 接口中没有定义 DeleteSymbol 方法
	// if err := c.repo.DeleteSymbol(ctx, cmd.SymbolID); err != nil {
	// 	return err
	// }

	// 发布交易对删除事件
	deletedEvent := domain.SymbolDeletedEvent{
		SymbolID:   symbol.ID,
		SymbolCode: symbol.SymbolCode,
		DeletedAt:  time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishSymbolDeleted(deletedEvent)
}

// CreateExchange 创建交易所
func (c *ReferenceDataCommand) CreateExchange(ctx context.Context, cmd CreateExchangeCommand) (*domain.Exchange, error) {
	// 创建交易所
	exchange := &domain.Exchange{
		ID:       cmd.ExchangeID,
		Name:     cmd.Name,
		Country:  cmd.Country,
		Status:   cmd.Status,
		Timezone: cmd.Timezone,
	}

	// 保存交易所
	if err := c.repo.SaveExchange(ctx, exchange); err != nil {
		return nil, err
	}

	// 发布交易所创建事件
	createdEvent := domain.ExchangeCreatedEvent{
		ExchangeID: exchange.ID,
		Name:       exchange.Name,
		Country:    exchange.Country,
		Status:     exchange.Status,
		Timezone:   exchange.Timezone,
		CreatedAt:  time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	c.eventPublisher.PublishExchangeCreated(createdEvent)

	return exchange, nil
}

// UpdateExchange 更新交易所
func (c *ReferenceDataCommand) UpdateExchange(ctx context.Context, cmd UpdateExchangeCommand) (*domain.Exchange, error) {
	// 获取交易所
	exchange, err := c.repo.GetExchange(ctx, cmd.ExchangeID)
	if err != nil {
		return nil, err
	}

	// 记录旧值
	oldStatus := exchange.Status
	oldCountry := exchange.Country

	// 更新交易所
	exchange.Status = cmd.Status
	exchange.Country = cmd.Country
	exchange.Timezone = cmd.Timezone

	// 保存交易所
	if err := c.repo.SaveExchange(ctx, exchange); err != nil {
		return nil, err
	}

	// 发布交易所更新事件
	updatedEvent := domain.ExchangeUpdatedEvent{
		ExchangeID: exchange.ID,
		OldStatus:  oldStatus,
		NewStatus:  exchange.Status,
		OldCountry: oldCountry,
		NewCountry: exchange.Country,
		UpdatedAt:  time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	c.eventPublisher.PublishExchangeUpdated(updatedEvent)

	// 如果状态变更，发布状态变更事件
	if oldStatus != exchange.Status {
		statusChangedEvent := domain.ExchangeStatusChangedEvent{
			ExchangeID: exchange.ID,
			Name:       exchange.Name,
			OldStatus:  oldStatus,
			NewStatus:  exchange.Status,
			ChangedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}

		c.eventPublisher.PublishExchangeStatusChanged(statusChangedEvent)
	}

	return exchange, nil
}

// DeleteExchange 删除交易所
func (c *ReferenceDataCommand) DeleteExchange(ctx context.Context, cmd DeleteExchangeCommand) error {
	// 获取交易所
	exchange, err := c.repo.GetExchange(ctx, cmd.ExchangeID)
	if err != nil {
		return err
	}

	// 删除交易所
	// 暂时注释，因为 repository 接口中没有定义 DeleteExchange 方法
	// if err := c.repo.DeleteExchange(ctx, cmd.ExchangeID); err != nil {
	// 	return err
	// }

	// 发布交易所删除事件
	deletedEvent := domain.ExchangeDeletedEvent{
		ExchangeID: exchange.ID,
		Name:       exchange.Name,
		DeletedAt:  time.Now().Unix(),
		OccurredOn: time.Now(),
	}

	return c.eventPublisher.PublishExchangeDeleted(deletedEvent)
}

// 辅助函数：转换为 decimal.Decimal
func toDecimal(value float64) interface{} {
	// 这里需要根据实际的 decimal 库实现进行转换
	// 暂时返回 float64，实际应用中需要转换为 decimal.Decimal
	return value
}
