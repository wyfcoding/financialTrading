package domain

import (
	"time"
)

// MetricCreatedEvent 指标创建事件
type MetricCreatedEvent struct {
	MetricName string
	Value      interface{}
	Tags       map[string]string
	Timestamp  int64
	OccurredOn time.Time
}

// AlertGeneratedEvent 告警生成事件
type AlertGeneratedEvent struct {
	AlertID     string
	RuleName    string
	Severity    string
	Message     string
	Source      string
	GeneratedAt int64
	OccurredOn  time.Time
}

// AlertStatusChangedEvent 告警状态变更事件
type AlertStatusChangedEvent struct {
	AlertID     string
	OldStatus   string
	NewStatus   string
	UpdatedAt   int64
	OccurredOn  time.Time
}

// SystemHealthChangedEvent 系统健康状态变更事件
type SystemHealthChangedEvent struct {
	ServiceName  string
	OldStatus    string
	NewStatus    string
	CPUUsage     float64
	MemoryUsage  float64
	Message      string
	LastChecked  int64
	OccurredOn   time.Time
}

// ExecutionAuditCreatedEvent 执行审计创建事件
type ExecutionAuditCreatedEvent struct {
	ID        string
	TradeID   string
	OrderID   string
	UserID    string
	Symbol    string
	Side      string
	Price     interface{}
	Quantity  interface{}
	Fee       interface{}
	Venue     string
	AlgoType  string
	Timestamp int64
	OccurredOn time.Time
}

// SpoofingDetectedEvent 哄骗检测事件
type SpoofingDetectedEvent struct {
	UserID       string
	Symbol       string
	OrderID      string
	DetectedAt   int64
	OccurredOn   time.Time
}

// MarketAnomalyDetectedEvent 市场异常检测事件
type MarketAnomalyDetectedEvent struct {
	Symbol      string
	AnomalyType string
	Details     map[string]interface{}
	DetectedAt  int64
	OccurredOn  time.Time
}
