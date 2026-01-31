package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/financialtrading/internal/position/infrastructure/messaging"
	"gorm.io/gorm"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishPositionCreated(event domain.PositionCreatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPositionUpdated(event domain.PositionUpdatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPositionClosed(event domain.PositionClosedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPositionPnLUpdated(event domain.PositionPnLUpdatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPositionCostMethodChanged(event domain.PositionCostMethodChangedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPositionFlip(event domain.PositionFlipEvent) error { return nil }

// PositionService 头寸服务门面，整合命令和查询服务
type PositionService struct {
	Command *PositionCommand
	Query   *PositionQueryService
}

// NewPositionService 构造函数
func NewPositionService(repo domain.PositionRepository, db interface{}) (*PositionService, error) {
	// 创建事件发布者
	var eventPublisher domain.EventPublisher
	if gormDB, ok := db.(*gorm.DB); ok {
		eventPublisher = messaging.NewOutboxEventPublisher(gormDB)
	} else {
		// 使用空实现作为降级方案
		eventPublisher = &mockEventPublisher{}
	}

	// 创建命令服务
	command := NewPositionCommand(repo, eventPublisher)

	// 创建查询服务
	query := NewPositionQueryService(repo)

	return &PositionService{
		Command: command,
		Query:   query,
	}, nil
}

// --- Command (Writes) ---

// UpdatePosition 更新头寸
func (s *PositionService) UpdatePosition(ctx context.Context, cmd UpdatePositionCommand) (*domain.Position, error) {
	return s.Command.UpdatePosition(ctx, cmd)
}

// ChangeCostMethod 变更成本计算方法
func (s *PositionService) ChangeCostMethod(ctx context.Context, cmd ChangeCostMethodCommand) error {
	return s.Command.ChangeCostMethod(ctx, cmd)
}

// ClosePosition 平仓
func (s *PositionService) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	// 这里应该是实际的平仓逻辑
	// 暂时返回 nil
	return nil
}

// TccTryFreeze TCC 尝试冻结
func (s *PositionService) TccTryFreeze(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	// 这里应该是实际的 TCC 尝试冻结逻辑
	// 暂时返回 nil
	return nil
}

// TccConfirmFreeze TCC 确认冻结
func (s *PositionService) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	// 这里应该是实际的 TCC 确认冻结逻辑
	// 暂时返回 nil
	return nil
}

// TccCancelFreeze TCC 取消冻结
func (s *PositionService) TccCancelFreeze(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	// 这里应该是实际的 TCC 取消冻结逻辑
	// 暂时返回 nil
	return nil
}

// SagaDeductFrozen SAGA 扣减冻结
func (s *PositionService) SagaDeductFrozen(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	// 这里应该是实际的 SAGA 扣减冻结逻辑
	// 暂时返回 nil
	return nil
}

// SagaRefundFrozen SAGA 退还冻结
func (s *PositionService) SagaRefundFrozen(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	// 这里应该是实际的 SAGA 退还冻结逻辑
	// 暂时返回 nil
	return nil
}

// SagaAddPosition SAGA 增加头寸
func (s *PositionService) SagaAddPosition(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	// 这里应该是实际的 SAGA 增加头寸逻辑
	// 暂时返回 nil
	return nil
}

// SagaSubPosition SAGA 减少头寸
func (s *PositionService) SagaSubPosition(ctx context.Context, barrier interface{}, userID string, symbol string, quantity decimal.Decimal) error {
	// 这里应该是实际的 SAGA 减少头寸逻辑
	// 暂时返回 nil
	return nil
}

// --- Query (Reads) ---

// GetPositions 获取用户所有头寸
func (s *PositionService) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	return s.Query.GetPositions(ctx, userID, limit, offset)
}

// GetPosition 获取单个头寸
func (s *PositionService) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	return s.Query.GetPosition(ctx, positionID)
}
