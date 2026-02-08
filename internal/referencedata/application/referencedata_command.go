package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue"
)

// ReferenceDataCommandService 处理参考数据相关的命令操作
// Writes 统一走 MySQL + Outbox 事件发布。
type ReferenceDataCommandService struct {
	repo      domain.ReferenceDataRepository
	publisher messagequeue.EventPublisher
}

// NewReferenceDataCommandService 创建新的命令服务
func NewReferenceDataCommandService(repo domain.ReferenceDataRepository, publisher messagequeue.EventPublisher) *ReferenceDataCommandService {
	return &ReferenceDataCommandService{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateSymbol 创建交易对
func (s *ReferenceDataCommandService) CreateSymbol(ctx context.Context, cmd CreateSymbolCommand) (*domain.Symbol, error) {
	symbolID := cmd.SymbolID
	if symbolID == "" {
		symbolID = fmt.Sprintf("SYM-%d", idgen.GenID())
	}

	symbol := &domain.Symbol{
		ID:             symbolID,
		BaseCurrency:   cmd.BaseCurrency,
		QuoteCurrency:  cmd.QuoteCurrency,
		ExchangeID:     cmd.ExchangeID,
		SymbolCode:     cmd.SymbolCode,
		Status:         cmd.Status,
		MinOrderSize:   decimal.NewFromFloat(cmd.MinOrderSize),
		PricePrecision: decimal.NewFromFloat(cmd.PricePrecision),
	}

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveSymbol(txCtx, symbol); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}

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
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SymbolCreatedEventType, symbol.ID, createdEvent)
	})
	if err != nil {
		return nil, err
	}

	return symbol, nil
}

// UpdateSymbol 更新交易对
func (s *ReferenceDataCommandService) UpdateSymbol(ctx context.Context, cmd UpdateSymbolCommand) (*domain.Symbol, error) {
	symbol, err := s.repo.GetSymbol(ctx, cmd.SymbolID)
	if err != nil {
		return nil, err
	}
	if symbol == nil {
		return nil, fmt.Errorf("symbol not found")
	}

	oldStatus := symbol.Status
	oldMinOrderSize := symbol.MinOrderSize.InexactFloat64()

	symbol.Status = cmd.Status
	symbol.MinOrderSize = decimal.NewFromFloat(cmd.MinOrderSize)
	symbol.PricePrecision = decimal.NewFromFloat(cmd.PricePrecision)

	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveSymbol(txCtx, symbol); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}

		updatedEvent := domain.SymbolUpdatedEvent{
			SymbolID:        symbol.ID,
			OldStatus:       oldStatus,
			NewStatus:       symbol.Status,
			OldMinOrderSize: oldMinOrderSize,
			NewMinOrderSize: cmd.MinOrderSize,
			UpdatedAt:       time.Now().Unix(),
			OccurredOn:      time.Now(),
		}
		if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SymbolUpdatedEventType, symbol.ID, updatedEvent); err != nil {
			return err
		}

		if oldStatus != symbol.Status {
			statusChangedEvent := domain.SymbolStatusChangedEvent{
				SymbolID:   symbol.ID,
				SymbolCode: symbol.SymbolCode,
				OldStatus:  oldStatus,
				NewStatus:  symbol.Status,
				ChangedAt:  time.Now().Unix(),
				OccurredOn: time.Now(),
			}
			if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SymbolStatusChangedEventType, symbol.ID, statusChangedEvent); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return symbol, nil
}

// DeleteSymbol 删除交易对
func (s *ReferenceDataCommandService) DeleteSymbol(ctx context.Context, cmd DeleteSymbolCommand) error {
	symbol, err := s.repo.GetSymbol(ctx, cmd.SymbolID)
	if err != nil {
		return err
	}
	if symbol == nil {
		return nil
	}

	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.DeleteSymbol(txCtx, symbol.ID); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		deletedEvent := domain.SymbolDeletedEvent{
			SymbolID:   symbol.ID,
			SymbolCode: symbol.SymbolCode,
			DeletedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SymbolDeletedEventType, symbol.ID, deletedEvent)
	})
}

// CreateExchange 创建交易所
func (s *ReferenceDataCommandService) CreateExchange(ctx context.Context, cmd CreateExchangeCommand) (*domain.Exchange, error) {
	exchangeID := cmd.ExchangeID
	if exchangeID == "" {
		exchangeID = fmt.Sprintf("EX-%d", idgen.GenID())
	}

	exchange := &domain.Exchange{
		ID:       exchangeID,
		Name:     cmd.Name,
		Country:  cmd.Country,
		Status:   cmd.Status,
		Timezone: cmd.Timezone,
	}

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveExchange(txCtx, exchange); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		createdEvent := domain.ExchangeCreatedEvent{
			ExchangeID: exchange.ID,
			Name:       exchange.Name,
			Country:    exchange.Country,
			Status:     exchange.Status,
			Timezone:   exchange.Timezone,
			CreatedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.ExchangeCreatedEventType, exchange.ID, createdEvent)
	})
	if err != nil {
		return nil, err
	}

	return exchange, nil
}

// UpdateExchange 更新交易所
func (s *ReferenceDataCommandService) UpdateExchange(ctx context.Context, cmd UpdateExchangeCommand) (*domain.Exchange, error) {
	exchange, err := s.repo.GetExchange(ctx, cmd.ExchangeID)
	if err != nil {
		return nil, err
	}
	if exchange == nil {
		return nil, fmt.Errorf("exchange not found")
	}

	oldStatus := exchange.Status
	oldCountry := exchange.Country

	exchange.Status = cmd.Status
	exchange.Country = cmd.Country
	exchange.Timezone = cmd.Timezone

	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.SaveExchange(txCtx, exchange); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}

		updatedEvent := domain.ExchangeUpdatedEvent{
			ExchangeID: exchange.ID,
			OldStatus:  oldStatus,
			NewStatus:  exchange.Status,
			OldCountry: oldCountry,
			NewCountry: exchange.Country,
			UpdatedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}
		if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.ExchangeUpdatedEventType, exchange.ID, updatedEvent); err != nil {
			return err
		}

		if oldStatus != exchange.Status {
			statusChangedEvent := domain.ExchangeStatusChangedEvent{
				ExchangeID: exchange.ID,
				Name:       exchange.Name,
				OldStatus:  oldStatus,
				NewStatus:  exchange.Status,
				ChangedAt:  time.Now().Unix(),
				OccurredOn: time.Now(),
			}
			if err := s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.ExchangeStatusChangedEventType, exchange.ID, statusChangedEvent); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return exchange, nil
}

// DeleteExchange 删除交易所
func (s *ReferenceDataCommandService) DeleteExchange(ctx context.Context, cmd DeleteExchangeCommand) error {
	exchange, err := s.repo.GetExchange(ctx, cmd.ExchangeID)
	if err != nil {
		return err
	}
	if exchange == nil {
		return nil
	}

	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.DeleteExchange(txCtx, exchange.ID); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		deletedEvent := domain.ExchangeDeletedEvent{
			ExchangeID: exchange.ID,
			Name:       exchange.Name,
			DeletedAt:  time.Now().Unix(),
			OccurredOn: time.Now(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.ExchangeDeletedEventType, exchange.ID, deletedEvent)
	})
}
