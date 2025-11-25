package domain

import (
	"context"
	"time"
)

// Metric 指标实体
type Metric struct {
	Name      string
	Value     float64
	Tags      map[string]string
	Timestamp time.Time
}

// SystemHealth 系统健康状态实体
type SystemHealth struct {
	ServiceName string
	Status      string // UP, DOWN, DEGRADED
	Message     string
	LastChecked time.Time
}

// MetricRepository 指标仓储接口
type MetricRepository interface {
	Save(ctx context.Context, metric *Metric) error
	GetMetrics(ctx context.Context, name string, startTime, endTime time.Time) ([]*Metric, error)
}

// SystemHealthRepository 系统健康仓储接口
type SystemHealthRepository interface {
	Save(ctx context.Context, health *SystemHealth) error
	GetLatestHealth(ctx context.Context, serviceName string) ([]*SystemHealth, error)
}
