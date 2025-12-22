// 包 持仓服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/position/domain"
	"github.com/wyfcoding/pkg/logging"
)

// PositionDTO 持仓 DTO
// 用于在应用层和接口层之间传输持仓数据
type PositionDTO struct {
	PositionID    string // 持仓 ID
	UserID        string // 用户 ID
	Symbol        string // 交易对
	Side          string // 买卖方向
	Quantity      string // 持仓数量
	EntryPrice    string // 开仓价格
	CurrentPrice  string // 当前价格
	UnrealizedPnL string // 未实现盈亏
	RealizedPnL   string // 已实现盈亏
	OpenedAt      int64  // 开仓时间戳
	ClosedAt      *int64 // 平仓时间戳 (可选)
	Status        string // 状态 (OPEN, CLOSED)
}

// PositionApplicationService 持仓应用服务
// 负责处理持仓查询、平仓等业务逻辑
type PositionApplicationService struct {
	positionRepo domain.PositionRepository // 持仓仓储接口
}

// NewPositionApplicationService 创建持仓应用服务
// positionRepo: 注入的持仓仓储实现
func NewPositionApplicationService(positionRepo domain.PositionRepository) *PositionApplicationService {
	return &PositionApplicationService{
		positionRepo: positionRepo,
	}
}

// GetPositions 获取用户持仓列表
func (pas *PositionApplicationService) GetPositions(ctx context.Context, userID string, limit, offset int) ([]*PositionDTO, int64, error) {
	// 验证输入
	if userID == "" {
		return nil, 0, fmt.Errorf("user_id is required")
	}

	// 获取持仓列表
	positions, total, err := pas.positionRepo.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		logging.Error(ctx, "Failed to get positions",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get positions: %w", err)
	}

	// 转换为 DTO 列表
	dtos := make([]*PositionDTO, 0, len(positions))
	for _, position := range positions {
		closedAt := int64(0)
		if position.ClosedAt != nil {
			closedAt = position.ClosedAt.Unix()
		}

		dtos = append(dtos, &PositionDTO{
			PositionID:    position.PositionID,
			UserID:        position.UserID,
			Symbol:        position.Symbol,
			Side:          position.Side,
			Quantity:      position.Quantity.String(),
			EntryPrice:    position.EntryPrice.String(),
			CurrentPrice:  position.CurrentPrice.String(),
			UnrealizedPnL: position.UnrealizedPnL.String(),
			RealizedPnL:   position.RealizedPnL.String(),
			OpenedAt:      position.OpenedAt.Unix(),
			ClosedAt:      &closedAt,
			Status:        position.Status,
		})
	}

	return dtos, total, nil
}

// GetPosition 获取持仓详情
func (pas *PositionApplicationService) GetPosition(ctx context.Context, positionID string) (*PositionDTO, error) {
	// 验证输入
	if positionID == "" {
		return nil, fmt.Errorf("position_id is required")
	}

	// 获取持仓
	position, err := pas.positionRepo.Get(ctx, positionID)
	if err != nil {
		logging.Error(ctx, "Failed to get position",
			"position_id", positionID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if position == nil {
		return nil, fmt.Errorf("position not found: %s", positionID)
	}

	// 转换为 DTO
	closedAt := int64(0)
	if position.ClosedAt != nil {
		closedAt = position.ClosedAt.Unix()
	}

	return &PositionDTO{
		PositionID:    position.PositionID,
		UserID:        position.UserID,
		Symbol:        position.Symbol,
		Side:          position.Side,
		Quantity:      position.Quantity.String(),
		EntryPrice:    position.EntryPrice.String(),
		CurrentPrice:  position.CurrentPrice.String(),
		UnrealizedPnL: position.UnrealizedPnL.String(),
		RealizedPnL:   position.RealizedPnL.String(),
		OpenedAt:      position.OpenedAt.Unix(),
		ClosedAt:      &closedAt,
		Status:        position.Status,
	}, nil
}

// ClosePosition 平仓
func (pas *PositionApplicationService) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	// 验证输入
	if positionID == "" || closePrice.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("invalid request parameters")
	}

	// 平仓
	if err := pas.positionRepo.Close(ctx, positionID, closePrice); err != nil {
		logging.Error(ctx, "Failed to close position",
			"position_id", positionID,
			"error", err,
		)
		return fmt.Errorf("failed to close position: %w", err)
	}

	logging.Debug(ctx, "Position closed successfully",
		"position_id", positionID,
		"close_price", closePrice.String(),
	)

	return nil
}
