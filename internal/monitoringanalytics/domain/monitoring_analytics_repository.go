package domain

import (
	"context"
)

// MetricRepository 指标仓储接口
type MetricRepository interface {
	// Save 保存指标数据
	Save(ctx context.Context, metric *Metric) error
	// GetMetrics 获取指标历史数据
	GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*Metric, error)
}

// SystemHealthRepository 系统健康仓储接口
type SystemHealthRepository interface {
	// Save 保存健康状态
	Save(ctx context.Context, health *SystemHealth) error
	// GetLatestHealth 获取服务最新的健康检查记录
	GetLatestHealth(ctx context.Context, serviceName string, limit int) ([]*SystemHealth, error)
}
