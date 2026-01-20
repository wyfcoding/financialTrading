// Package mysql 提供了监控分析服务指标与系统健康仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"encoding/json"
	"time"

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

func (r *metricRepositoryImpl) GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*domain.TradeMetric, error) {
	var metrics []*domain.TradeMetric
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, startTime, endTime).
		Order("timestamp asc").
		Find(&metrics).Error
	return metrics, err
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

// AlertModel 告警数据库模型
type AlertModel struct {
	gorm.Model
	AlertID   string `gorm:"column:alert_id;type:varchar(32);uniqueIndex;not null"`
	RuleName  string `gorm:"column:rule_name;type:varchar(100);not null"`
	Severity  string `gorm:"column:severity;type:varchar(20)"`
	Message   string `gorm:"column:message;type:text"`
	Source    string `gorm:"column:source;type:varchar(50)"`
	Status    string `gorm:"column:status;type:varchar(20)"`
	Timestamp int64  `gorm:"column:timestamp;type:bigint;index"`
}

func (AlertModel) TableName() string { return "analytics_alerts" }

type alertRepositoryImpl struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) domain.AlertRepository {
	return &alertRepositoryImpl{db: db}
}

func (r *alertRepositoryImpl) Save(ctx context.Context, a *domain.Alert) error {
	m := &AlertModel{
		Model:     a.Model,
		AlertID:   a.AlertID,
		RuleName:  a.RuleName,
		Severity:  a.Severity,
		Message:   a.Message,
		Source:    a.Source,
		Status:    a.Status,
		Timestamp: a.GeneratedAt,
	}
	err := r.db.WithContext(ctx).Save(m).Error
	if err == nil {
		a.Model = m.Model
	}
	return err
}

func (r *alertRepositoryImpl) GetAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	var models []AlertModel
	if err := r.db.WithContext(ctx).Order("timestamp desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Alert, len(models))
	for i, m := range models {
		res[i] = &domain.Alert{
			Model:       m.Model,
			AlertID:     m.AlertID,
			RuleName:    m.RuleName,
			Severity:    m.Severity,
			Message:     m.Message,
			Source:      m.Source,
			Status:      m.Status,
			GeneratedAt: m.Timestamp,
		}
	}
	return res, nil
}
