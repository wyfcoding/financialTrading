package domain

import (
	"context"
)

// SettlementRepository 是清算记录的仓储接口
type SettlementRepository interface {
	// Save 保存或更新清算记录
	Save(ctx context.Context, settlement *Settlement) error
	// Get 根据 SettlementID 获取一个清算记录
	Get(ctx context.Context, settlementID string) (*Settlement, error)
	// GetByUser 分页获取指定用户的清算历史记录
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Settlement, int64, error)
	// GetByTrade 根据 TradeID 获取一个清算记录
	GetByTrade(ctx context.Context, tradeID string) (*Settlement, error)
}

// EODClearingRepository 是日终清算任务的仓储接口
type EODClearingRepository interface {
	// Save 保存或更新日终清算任务
	Save(ctx context.Context, clearing *EODClearing) error
	// Get 根据 ClearingID 获取一个日终清算任务
	Get(ctx context.Context, clearingID string) (*EODClearing, error)
	// GetLatest 获取最新的一次日终清算任务
	GetLatest(ctx context.Context) (*EODClearing, error)
	// Update 显式更新日终清算任务信息
	Update(ctx context.Context, clearing *EODClearing) error
}

// MarginRequirementRepository 是保证金要求的仓储接口
type MarginRequirementRepository interface {
	// Save 保存或更新保证金要求
	Save(ctx context.Context, margin *MarginRequirement) error
	// GetBySymbol 根据交易对获取保证金要求
	GetBySymbol(ctx context.Context, symbol string) (*MarginRequirement, error)
}
