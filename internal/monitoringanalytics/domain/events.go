package domain

import "time"

const (
	MetricCreatedEventType         = "monitoring.metric.created"
	AlertGeneratedEventType        = "monitoring.alert.generated"
	AlertStatusChangedEventType    = "monitoring.alert.status.changed"
	SystemHealthChangedEventType   = "monitoring.systemhealth.changed"
	ExecutionAuditCreatedEventType = "monitoring.audit.created"
	SpoofingDetectedEventType      = "monitoring.spoofing.detected"
	MarketAnomalyDetectedEventType = "monitoring.anomaly.detected"
)

// MetricCreatedEvent 指标创建事件
type MetricCreatedEvent struct {
	MetricName string            `json:"metric_name"`
	Value      interface{}       `json:"value"`
	Tags       map[string]string `json:"tags"`
	Timestamp  int64             `json:"timestamp"`
	OccurredOn time.Time         `json:"occurred_on"`
}

// AlertGeneratedEvent 告警生成事件
type AlertGeneratedEvent struct {
	AlertID     string    `json:"alert_id"`
	RuleName    string    `json:"rule_name"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	GeneratedAt int64     `json:"generated_at"`
	OccurredOn  time.Time `json:"occurred_on"`
}

// AlertStatusChangedEvent 告警状态变更事件
type AlertStatusChangedEvent struct {
	AlertID    string    `json:"alert_id"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	UpdatedAt  int64     `json:"updated_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// SystemHealthChangedEvent 系统健康状态变更事件
type SystemHealthChangedEvent struct {
	ServiceName string    `json:"service_name"`
	OldStatus   string    `json:"old_status"`
	NewStatus   string    `json:"new_status"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	Message     string    `json:"message"`
	LastChecked int64     `json:"last_checked"`
	OccurredOn  time.Time `json:"occurred_on"`
}

// ExecutionAuditCreatedEvent 执行审计创建事件
type ExecutionAuditCreatedEvent struct {
	ID         string      `json:"id"`
	TradeID    string      `json:"trade_id"`
	OrderID    string      `json:"order_id"`
	UserID     string      `json:"user_id"`
	Symbol     string      `json:"symbol"`
	Side       string      `json:"side"`
	Price      interface{} `json:"price"`
	Quantity   interface{} `json:"quantity"`
	Fee        interface{} `json:"fee"`
	Venue      string      `json:"venue"`
	AlgoType   string      `json:"algo_type"`
	Timestamp  int64       `json:"timestamp"`
	OccurredOn time.Time   `json:"occurred_on"`
}

// SpoofingDetectedEvent 哄骗检测事件
type SpoofingDetectedEvent struct {
	UserID     string    `json:"user_id"`
	Symbol     string    `json:"symbol"`
	OrderID    string    `json:"order_id"`
	DetectedAt int64     `json:"detected_at"`
	OccurredOn time.Time `json:"occurred_on"`
}

// MarketAnomalyDetectedEvent 市场异常检测事件
type MarketAnomalyDetectedEvent struct {
	Symbol      string                 `json:"symbol"`
	AnomalyType string                 `json:"anomaly_type"`
	Details     map[string]interface{} `json:"details"`
	DetectedAt  int64                  `json:"detected_at"`
	OccurredOn  time.Time              `json:"occurred_on"`
}
