package mysql

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"gorm.io/gorm"
)

// MetricModel 指标数据库模型
type MetricModel struct {
	gorm.Model
	Name      string `gorm:"column:name;type:varchar(100);index;not null"`
	Value     string `gorm:"column:value;type:decimal(32,18);not null"`
	TagsJSON  string `gorm:"column:tags;type:text"`
	Timestamp int64  `gorm:"column:timestamp;type:bigint;index;not null"`
}

func (MetricModel) TableName() string { return "analytics_metrics" }

// SystemHealthModel 系统健康数据库模型
type SystemHealthModel struct {
	gorm.Model
	ServiceName string  `gorm:"column:service_name;type:varchar(100);index;not null"`
	Status      string  `gorm:"column:status;type:varchar(20);not null"`
	CPUUsage    float64 `gorm:"column:cpu_usage;type:decimal(5,2)"`
	MemoryUsage float64 `gorm:"column:memory_usage;type:decimal(5,2)"`
	Message     string  `gorm:"column:message;type:text"`
	LastChecked int64   `gorm:"column:last_checked;type:bigint;not null"`
}

func (SystemHealthModel) TableName() string { return "analytics_system_health" }

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

// TradeMetricModel 交易指标数据库模型
type TradeMetricModel struct {
	gorm.Model
	Symbol       string    `gorm:"column:symbol;type:varchar(20);index"`
	MetricType   string    `gorm:"column:metric_type;type:varchar(20)"`
	Timestamp    time.Time `gorm:"column:timestamp;index"`
	TotalVolume  float64   `gorm:"column:total_volume;type:decimal(20,8)"`
	TradeCount   int       `gorm:"column:trade_count;type:int"`
	AveragePrice float64   `gorm:"column:average_price;type:decimal(20,8)"`
}

func (TradeMetricModel) TableName() string { return "trade_metrics" }

// mapping helpers

func toMetricModel(m *domain.Metric) (*MetricModel, error) {
	if m == nil {
		return nil, nil
	}
	tagsJSON := m.TagsJSON
	if tagsJSON == "" && len(m.Tags) > 0 {
		raw, err := json.Marshal(m.Tags)
		if err != nil {
			return nil, err
		}
		tagsJSON = string(raw)
	}
	return &MetricModel{
		Model: gorm.Model{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		Name:      m.Name,
		Value:     m.Value.String(),
		TagsJSON:  tagsJSON,
		Timestamp: m.Timestamp,
	}, nil
}

func toMetric(m *MetricModel) (*domain.Metric, error) {
	if m == nil {
		return nil, nil
	}
	val, err := decimal.NewFromString(m.Value)
	if err != nil {
		return nil, err
	}
	var tags map[string]string
	if m.TagsJSON != "" {
		if err := json.Unmarshal([]byte(m.TagsJSON), &tags); err != nil {
			return nil, err
		}
	}
	return &domain.Metric{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Name:      m.Name,
		Value:     val,
		Tags:      tags,
		TagsJSON:  m.TagsJSON,
		Timestamp: m.Timestamp,
	}, nil
}

func toHealthModel(h *domain.SystemHealth) *SystemHealthModel {
	if h == nil {
		return nil
	}
	return &SystemHealthModel{
		Model: gorm.Model{
			ID:        h.ID,
			CreatedAt: h.CreatedAt,
			UpdatedAt: h.UpdatedAt,
		},
		ServiceName: h.ServiceName,
		Status:      h.Status,
		CPUUsage:    h.CPUUsage,
		MemoryUsage: h.MemoryUsage,
		Message:     h.Message,
		LastChecked: h.LastChecked,
	}
}

func toHealth(m *SystemHealthModel) *domain.SystemHealth {
	if m == nil {
		return nil
	}
	return &domain.SystemHealth{
		ID:          m.ID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		ServiceName: m.ServiceName,
		Status:      m.Status,
		CPUUsage:    m.CPUUsage,
		MemoryUsage: m.MemoryUsage,
		Message:     m.Message,
		LastChecked: m.LastChecked,
	}
}

func toAlertModel(a *domain.Alert) *AlertModel {
	if a == nil {
		return nil
	}
	return &AlertModel{
		Model: gorm.Model{
			ID:        a.ID,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		},
		AlertID:   a.AlertID,
		RuleName:  a.RuleName,
		Severity:  a.Severity,
		Message:   a.Message,
		Source:    a.Source,
		Status:    a.Status,
		Timestamp: a.GeneratedAt,
	}
}

func toAlert(m *AlertModel) *domain.Alert {
	if m == nil {
		return nil
	}
	return &domain.Alert{
		ID:          m.ID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		AlertID:     m.AlertID,
		RuleName:    m.RuleName,
		Severity:    m.Severity,
		Message:     m.Message,
		Source:      m.Source,
		Status:      m.Status,
		GeneratedAt: m.Timestamp,
	}
}

func toTradeMetric(m *TradeMetricModel) *domain.TradeMetric {
	if m == nil {
		return nil
	}
	return &domain.TradeMetric{
		ID:           m.ID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Symbol:       m.Symbol,
		MetricType:   m.MetricType,
		Timestamp:    m.Timestamp,
		TotalVolume:  m.TotalVolume,
		TradeCount:   m.TradeCount,
		AveragePrice: m.AveragePrice,
	}
}
