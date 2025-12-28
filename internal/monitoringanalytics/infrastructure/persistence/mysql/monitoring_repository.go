// Package mysql 提供了监控分析服务指标与系统健康仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"encoding/json"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MetricModel 指标数据库模型
type MetricModel struct {
	gorm.Model
	Name      string `gorm:"column:name;type:varchar(100);index;not null"`
	Value     string `gorm:"column:value;type:decimal(32,18);not null"`
	Tags      string `gorm:"column:tags;type:text"`
	Timestamp int64  `gorm:"column:timestamp;type:bigint;index;not null"`
}

func (MetricModel) TableName() string { return "analytics_metrics" }

// SystemHealthModel 系统健康数据库模型
type SystemHealthModel struct {
	gorm.Model
	ServiceName string `gorm:"column:service_name;type:varchar(100);index;not null"`
	Status      string `gorm:"column:status;type:varchar(20);not null"`
	Message     string `gorm:"column:message;type:text"`
	LastChecked int64  `gorm:"column:last_checked;type:bigint;not null"`
}

func (SystemHealthModel) TableName() string { return "analytics_system_health" }

type metricRepositoryImpl struct {
	db *gorm.DB
}

func NewMetricRepository(db *gorm.DB) domain.MetricRepository {
	return &metricRepositoryImpl{db: db}
}

func (r *metricRepositoryImpl) Save(ctx context.Context, m *domain.Metric) error {
	tags, err := json.Marshal(m.Tags)
	if err != nil {
		return err
	}
	model := &MetricModel{
		Model:     m.Model,
		Name:      m.Name,
		Value:     m.Value.String(),
		Tags:      string(tags),
		Timestamp: m.Timestamp,
	}
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(model).Error
	if err == nil {
		m.Model = model.Model
	}
	return err
}

func (r *metricRepositoryImpl) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	var models []MetricModel
	if err := r.db.WithContext(ctx).Where("name = ? AND timestamp >= ? AND timestamp <= ?", name, startTime, endTime).Order("timestamp asc").Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Metric, len(models))
	for i, m := range models {
		dm, err := r.metricToDomain(&m)
		if err != nil {
			return nil, err
		}
		res[i] = dm
	}
	return res, nil
}

func (r *metricRepositoryImpl) metricToDomain(m *MetricModel) (*domain.Metric, error) {
	val, err := decimal.NewFromString(m.Value)
	if err != nil {
		return nil, err
	}
	var tags map[string]string
	if err := json.Unmarshal([]byte(m.Tags), &tags); err != nil {
		return nil, err
	}
	return &domain.Metric{
		Model:     m.Model,
		Name:      m.Name,
		Value:     val,
		Tags:      tags,
		TagsJSON:  m.Tags,
		Timestamp: m.Timestamp,
	}, nil
}

type systemHealthRepositoryImpl struct {
	db *gorm.DB
}

func NewSystemHealthRepository(db *gorm.DB) domain.SystemHealthRepository {
	return &systemHealthRepositoryImpl{db: db}
}

func (r *systemHealthRepositoryImpl) Save(ctx context.Context, h *domain.SystemHealth) error {
	model := &SystemHealthModel{
		Model:       h.Model,
		ServiceName: h.ServiceName,
		Status:      h.Status,
		Message:     h.Message,
		LastChecked: h.LastChecked,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(model).Error
	if err == nil {
		h.Model = model.Model
	}
	return err
}

func (r *systemHealthRepositoryImpl) GetLatestHealth(ctx context.Context, serviceName string, limit int) ([]*domain.SystemHealth, error) {
	var models []SystemHealthModel
	query := r.db.WithContext(ctx)
	if serviceName != "" {
		query = query.Where("service_name = ?", serviceName)
	}
	if err := query.Order("last_checked desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.SystemHealth, len(models))
	for i, m := range models {
		res[i] = r.healthToDomain(&m)
	}
	return res, nil
}

func (r *systemHealthRepositoryImpl) healthToDomain(m *SystemHealthModel) *domain.SystemHealth {
	return &domain.SystemHealth{
		Model:       m.Model,
		ServiceName: m.ServiceName,
		Status:      m.Status,
		Message:     m.Message,
		LastChecked: m.LastChecked,
	}
}
