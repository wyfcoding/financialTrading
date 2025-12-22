// 包 基础设施层实现
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wyfcoding/financialTrading/internal/monitoring-analytics/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// MetricModel 指标数据库模型
// 对应数据库中的 metrics 表
type MetricModel struct {
	gorm.Model
	Name      string    `gorm:"column:name;type:varchar(100);index;not null;comment:指标名称"`
	Value     float64   `gorm:"column:value;type:decimal(20,8);not null;comment:指标值"`
	Tags      string    `gorm:"column:tags;type:text;comment:标签JSON"`
	Timestamp time.Time `gorm:"column:timestamp;index;not null;comment:时间戳"`
}

// 指定表名
func (MetricModel) TableName() string {
	return "metrics"
}

// 将数据库模型转换为领域实体
func (m *MetricModel) ToDomain() *domain.Metric {
	var tags map[string]string
	_ = json.Unmarshal([]byte(m.Tags), &tags)
	return &domain.Metric{
		Model:     m.Model,
		Name:      m.Name,
		Value:     m.Value,
		Tags:      tags,
		TagsJSON:  m.Tags,
		Timestamp: m.Timestamp,
	}
}

// MetricRepositoryImpl 指标仓储实现
type MetricRepositoryImpl struct {
	db *gorm.DB
}

// NewMetricRepository 创建指标仓储实例
func NewMetricRepository(db *gorm.DB) domain.MetricRepository {
	return &MetricRepositoryImpl{db: db}
}

// Save 保存指标
func (r *MetricRepositoryImpl) Save(ctx context.Context, metric *domain.Metric) error {
	tagsJSON, _ := json.Marshal(metric.Tags)
	model := &MetricModel{
		Model:     metric.Model,
		Name:      metric.Name,
		Value:     metric.Value,
		Tags:      string(tagsJSON),
		Timestamp: metric.Timestamp,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save metric",
			"name", metric.Name,
			"error", err,
		)
		return fmt.Errorf("failed to save metric: %w", err)
	}

	metric.Model = model.Model
	metric.TagsJSON = string(tagsJSON)
	return nil
}

// GetMetrics 获取指标列表
func (r *MetricRepositoryImpl) GetMetrics(ctx context.Context, name string, startTime, endTime time.Time) ([]*domain.Metric, error) {
	var models []MetricModel
	if err := r.db.WithContext(ctx).Where("name = ? AND timestamp BETWEEN ? AND ?", name, startTime, endTime).Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get metrics",
			"name", name,
			"start_time", startTime,
			"end_time", endTime,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics := make([]*domain.Metric, len(models))
	for i, m := range models {
		metrics[i] = m.ToDomain()
	}
	return metrics, nil
}

// SystemHealthModel 系统健康数据库模型
type SystemHealthModel struct {
	gorm.Model
	ServiceName string    `gorm:"column:service_name;type:varchar(100);index;not null;comment:服务名称"`
	Status      string    `gorm:"column:status;type:varchar(20);not null;comment:状态"`
	Message     string    `gorm:"column:message;type:text;comment:消息"`
	LastChecked time.Time `gorm:"column:last_checked;not null;comment:最后检查时间"`
}

// 指定表名
func (SystemHealthModel) TableName() string {
	return "system_health"
}

// ToDomain 转换为领域实体
func (m *SystemHealthModel) ToDomain() *domain.SystemHealth {
	return &domain.SystemHealth{
		Model:       m.Model,
		ServiceName: m.ServiceName,
		Status:      m.Status,
		Message:     m.Message,
		LastChecked: m.LastChecked,
	}
}

// SystemHealthRepositoryImpl 系统健康仓储实现
type SystemHealthRepositoryImpl struct {
	db *gorm.DB
}

// NewSystemHealthRepository 创建系统健康仓储实例
func NewSystemHealthRepository(db *gorm.DB) domain.SystemHealthRepository {
	return &SystemHealthRepositoryImpl{db: db}
}

// Save 保存系统健康状态
func (r *SystemHealthRepositoryImpl) Save(ctx context.Context, health *domain.SystemHealth) error {
	model := &SystemHealthModel{
		Model:       health.Model,
		ServiceName: health.ServiceName,
		Status:      health.Status,
		Message:     health.Message,
		LastChecked: health.LastChecked,
	}
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save system health",
			"service_name", health.ServiceName,
			"error", err,
		)
		return fmt.Errorf("failed to save system health: %w", err)
	}

	health.Model = model.Model
	return nil
}

// GetLatestHealth 获取最新系统健康状态
func (r *SystemHealthRepositoryImpl) GetLatestHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	var models []SystemHealthModel
	query := r.db.WithContext(ctx)
	if serviceName != "" {
		query = query.Where("service_name = ?", serviceName)
	}

	// 获取每个服务的最新状态（简化实现：直接获取所有记录，实际应分组取最新）
	// 这里为了演示简单，只获取最近的100条
	if err := query.Order("last_checked desc").Limit(100).Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get latest health",
			"service_name", serviceName,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get latest health: %w", err)
	}

	healths := make([]*domain.SystemHealth, len(models))
	for i, m := range models {
		healths[i] = m.ToDomain()
	}
	return healths, nil
}
