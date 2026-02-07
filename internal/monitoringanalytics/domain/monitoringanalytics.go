package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Metric 指标实体
// 代表一个时间点上的监控指标数据
type Metric struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Name 指标名称
	Name string `json:"name"`
	// Value 指标值
	Value decimal.Decimal `json:"value"`
	// Tags 标签 (内存中 Map 表示)
	Tags map[string]string `json:"tags"`
	// TagsJSON 标签 (数据库存储 JSON 字符串)
	TagsJSON string `json:"tags_json"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp"`
}

// SystemHealth 系统健康状态实体
type SystemHealth struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ServiceName string    `json:"service_name"`
	Status      string    `json:"status"` // UP, DOWN, DEGRADED
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	Message     string    `json:"message"`
	// LastChecked 上次检查时间
	LastChecked int64 `json:"last_checked"`
}

// Alert 告警实体
type Alert struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	AlertID     string    `json:"alert_id"`
	RuleName    string    `json:"rule_name"`
	Severity    string    `json:"severity"` // INFO, WARNING, CRITICAL
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	GeneratedAt int64     `json:"generated_at"`
	Status      string    `json:"status"` // NEW, ACKNOWLEDGED, RESOLVED
}

func (a *Alert) Timestamp() int64 {
	return a.GeneratedAt
}

// ExecutionAudit 审计流水实体 (ClickHouse 优化)
type ExecutionAudit struct {
	ID        string          `json:"id"`
	TradeID   string          `json:"trade_id"`
	OrderID   string          `json:"order_id"`
	UserID    string          `json:"user_id"`
	Symbol    string          `json:"symbol"`
	Side      string          `json:"side"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"quantity"`
	Fee       decimal.Decimal `json:"fee"`
	Venue     string          `json:"venue"`
	AlgoType  string          `json:"algo_type"`
	Timestamp int64           `json:"timestamp"`
}

// End of domain file
