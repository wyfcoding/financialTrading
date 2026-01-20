package domain

import (
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
type SystemHealth struct {
	gorm.Model
	ServiceName string  `gorm:"column:service_name;type:varchar(50);not null;index"`
	Status      string  `gorm:"column:status;type:varchar(20);not null"` // UP, DOWN, DEGRADED
	CPUUsage    float64 `gorm:"column:cpu_usage;type:decimal(5,2)"`
	MemoryUsage float64 `gorm:"column:memory_usage;type:decimal(5,2)"`
	Message     string  `gorm:"column:message;type:text"`
	// LastChecked 上次检查时间
	LastChecked int64 `gorm:"column:last_checked;type:bigint;not null"`
}

// Alert 告警实体
type Alert struct {
	gorm.Model
	AlertID     string `gorm:"column:alert_id;type:varchar(32);uniqueIndex;not null"`
	RuleName    string `gorm:"column:rule_name;type:varchar(100);not null"`
	Severity    string `gorm:"column:severity;type:varchar(20);not null"` // INFO, WARNING, CRITICAL
	Message     string `gorm:"column:message;type:text;not null"`
	Source      string `gorm:"column:source;type:varchar(50)"`
	GeneratedAt int64  `gorm:"column:generated_at;type:bigint;not null"`
	Status      string `gorm:"column:status;type:varchar(20);default:'NEW'"` // NEW, ACKNOWLEDGED, RESOLVED
}

func (a *Alert) Timestamp() int64 {
	return a.GeneratedAt
}

// ExecutionAudit 审计流水实体 (ClickHouse 优化)
type ExecutionAudit struct {
	ID        string          `gorm:"primaryKey;type:varchar(32)" json:"id"`
	TradeID   string          `gorm:"index;type:varchar(32)" json:"trade_id"`
	OrderID   string          `gorm:"index;type:varchar(32)" json:"order_id"`
	UserID    string          `gorm:"index;type:varchar(64)" json:"user_id"`
	Symbol    string          `gorm:"index;type:varchar(20)" json:"symbol"`
	Side      string          `gorm:"type:varchar(10)" json:"side"`
	Price     decimal.Decimal `gorm:"type:decimal(32,18)" json:"price"`
	Quantity  decimal.Decimal `gorm:"type:decimal(32,18)" json:"quantity"`
	Fee       decimal.Decimal `gorm:"type:decimal(32,18)" json:"fee"`
	Venue     string          `gorm:"type:varchar(20)" json:"venue"`
	AlgoType  string          `gorm:"type:varchar(20)" json:"algo_type"`
	Timestamp int64           `gorm:"index;type:bigint" json:"timestamp"`
}

// End of domain file
