// 包 监控分析服务的领域模型
package domain

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Metric 指标实体
// 代表一个时间点上的监控指标数据
type Metric struct {
	gorm.Model
	// Name 指标名称
	Name string `gorm:"column:name;type:varchar(100);index;not null"`
	// Value 指标值
	Value decimal.Decimal `gorm:"column:value;type:decimal(32,18);not null"`
	// Tags 标签 (内存中 Map 表示)
	Tags map[string]string `gorm:"-"`
	// TagsJSON 标签 (数据库存储 JSON 字符串)
	TagsJSON string `gorm:"column:tags;type:text"`
	// Timestamp 时间戳
	Timestamp int64 `gorm:"column:timestamp;type:bigint;index;not null"`
}

// SystemHealth 系统健康状态实体
// 记录各个微服务的健康检查结果
type SystemHealth struct {
	gorm.Model
	// ServiceName 服务名称
	ServiceName string `gorm:"column:service_name;type:varchar(100);index;not null"`
	// Status 状态: UP(正常), DOWN(停止), DEGRADED(降级)
	Status string `gorm:"column:status;type:varchar(20);not null"`
	// Message 详细消息
	Message string `gorm:"column:message;type:text"`
	// LastChecked 上次检查时间
	LastChecked int64 `gorm:"column:last_checked;type:bigint;not null"`
}

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
