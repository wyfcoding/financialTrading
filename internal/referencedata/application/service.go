package application

import (
	"context"

	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"github.com/wyfcoding/financialtrading/internal/referencedata/infrastructure/messaging"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishSymbolCreated(event domain.SymbolCreatedEvent) error { return nil }
func (m *mockEventPublisher) PublishSymbolUpdated(event domain.SymbolUpdatedEvent) error { return nil }
func (m *mockEventPublisher) PublishSymbolDeleted(event domain.SymbolDeletedEvent) error { return nil }
func (m *mockEventPublisher) PublishSymbolStatusChanged(event domain.SymbolStatusChangedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishExchangeCreated(event domain.ExchangeCreatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishExchangeUpdated(event domain.ExchangeUpdatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishExchangeDeleted(event domain.ExchangeDeletedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishExchangeStatusChanged(event domain.ExchangeStatusChangedEvent) error {
	return nil
}

// ReferenceDataService 参考数据服务门面，整合命令和查询服务
type ReferenceDataService struct {
	Command *ReferenceDataCommand
	Query   *ReferenceDataQueryService
}

// NewReferenceDataService 构造函数
func NewReferenceDataService(repo domain.ReferenceDataRepository, db interface{}) (*ReferenceDataService, error) {
	// 创建事件发布者
	var eventPublisher domain.EventPublisher
	if gormDB, ok := db.(*gorm.DB); ok {
		eventPublisher = messaging.NewOutboxEventPublisher(gormDB)
	} else {
		// 使用空实现作为降级方案
		eventPublisher = &mockEventPublisher{}
	}

	// 创建命令服务
	command := NewReferenceDataCommand(repo, eventPublisher)

	// 创建查询服务
	query := NewReferenceDataQueryService(repo)

	return &ReferenceDataService{
		Command: command,
		Query:   query,
	}, nil
}

// --- Command (Writes) ---

// CreateSymbol 创建交易对
func (s *ReferenceDataService) CreateSymbol(ctx context.Context, cmd CreateSymbolCommand) (*domain.Symbol, error) {
	return s.Command.CreateSymbol(ctx, cmd)
}

// UpdateSymbol 更新交易对
func (s *ReferenceDataService) UpdateSymbol(ctx context.Context, cmd UpdateSymbolCommand) (*domain.Symbol, error) {
	return s.Command.UpdateSymbol(ctx, cmd)
}

// DeleteSymbol 删除交易对
func (s *ReferenceDataService) DeleteSymbol(ctx context.Context, cmd DeleteSymbolCommand) error {
	return s.Command.DeleteSymbol(ctx, cmd)
}

// CreateExchange 创建交易所
func (s *ReferenceDataService) CreateExchange(ctx context.Context, cmd CreateExchangeCommand) (*domain.Exchange, error) {
	return s.Command.CreateExchange(ctx, cmd)
}

// UpdateExchange 更新交易所
func (s *ReferenceDataService) UpdateExchange(ctx context.Context, cmd UpdateExchangeCommand) (*domain.Exchange, error) {
	return s.Command.UpdateExchange(ctx, cmd)
}

// DeleteExchange 删除交易所
func (s *ReferenceDataService) DeleteExchange(ctx context.Context, cmd DeleteExchangeCommand) error {
	return s.Command.DeleteExchange(ctx, cmd)
}

// --- Query (Reads) ---

// GetSymbol 获取单个交易对
func (s *ReferenceDataService) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	return s.Query.GetSymbol(ctx, id)
}

// ListSymbols 列表查询交易对
func (s *ReferenceDataService) ListSymbols(ctx context.Context, exchangeID, status string, limit int, offset int) ([]*domain.Symbol, error) {
	return s.Query.ListSymbols(ctx, exchangeID, status, limit, offset)
}

// GetExchange 获取交易所信息
func (s *ReferenceDataService) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	return s.Query.GetExchange(ctx, id)
}

// ListExchanges 交易所列表
func (s *ReferenceDataService) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	return s.Query.ListExchanges(ctx, limit, offset)
}
