package domain

import (
	"context"
)

// MarginRepository 保证金仓储接口
type MarginRepository interface {
	SaveAccount(ctx context.Context, account *MarginAccount) error
	FindAccountByUserID(ctx context.Context, userID uint64) (*MarginAccount, error)
	// 更多方法...
}

// MarginAccount 保证金账户领域对象
type MarginAccount struct {
	UserID            uint64  `json:"user_id"`
	TotalEquity       float64 `json:"total_equity"`
	UsedMargin        float64 `json:"used_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
}
