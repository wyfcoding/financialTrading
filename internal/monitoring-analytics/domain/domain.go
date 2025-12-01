// Package domain 包含监控分析服务的领域模型
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Metric 指标实体
// 代表一个时间点上的监控指标数据
type Metric struct {
	gorm.Model
	// 指标名称
	Name string `gorm:"column:name;type:varchar(100);index;not null" json:"name"`
	// 指标值
	Value float64 `gorm:"column:value;type:decimal(20,8);not null" json:"value"`
	// 标签 (内存中 Map 表示)
	Tags map[string]string `gorm:"-" json:"tags"`
	// 标签 (数据库存储 JSON 字符串)
	TagsJSON string `gorm:"column:tags;type:text" json:"-"`
	// 时间戳
	Timestamp time.Time `gorm:"column:timestamp;index;not null" json:"timestamp"`
}

// SystemHealth 系统健康状态实体
// 记录各个微服务的健康检查结果
type SystemHealth struct {
	gorm.Model
	ServiceName string    `gorm:"column:service_name;type:varchar(100);index;not null" json:"service_name"`
	Status      string    `gorm:"column:status;type:varchar(20);not null" json:"status"` // UP, DOWN, DEGRADED
	Message     string    `gorm:"column:message;type:text" json:"message"`
	LastChecked time.Time `gorm:"column:last_checked;not null" json:"last_checked"`
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
