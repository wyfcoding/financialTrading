package domain

import "context"

// ExecutionRepository 是执行记录的仓储接口
type ExecutionRepository interface {
	// Save 保存或更新执行记录
	Save(ctx context.Context, execution *Execution) error
	// Get 根据 ExecutionID 获取一个执行记录
	Get(ctx context.Context, executionID string) (*Execution, error)
	// GetByOrder 获取指定订单的所有执行记录
	GetByOrder(ctx context.Context, orderID string) ([]*Execution, error)
	// GetByUser 分页获取指定用户的执行历史记录
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Execution, int64, error)
}
