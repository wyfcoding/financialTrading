package domain

import (
	"context"
	"time"
)

// MetricRepository 定义指标数据的存储接口
type MetricRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, m *Metric) error
	GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*Metric, error)
	GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*TradeMetric, error)
}

// SystemHealthRepository 定义系统健康状态的存储接口
type SystemHealthRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, h *SystemHealth) error
	GetLatestHealth(ctx context.Context, serviceName string, limit int) ([]*SystemHealth, error)
}

type AlertRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	Save(ctx context.Context, alert *Alert) error
	UpdateStatus(ctx context.Context, alertID, status string) error
	GetAlerts(ctx context.Context, limit int) ([]*Alert, error)
}

// ExecutionAuditRepository 定义 ClickHouse 审计流水存储接口
type ExecutionAuditRepository interface {
	BatchSave(ctx context.Context, audits []*ExecutionAudit) error
	Query(ctx context.Context, userID, symbol string, startTime, endTime int64) ([]*ExecutionAudit, error)
}

// AuditESRepository 提供审计流水的全文检索与复杂查询能力
type AuditESRepository interface {
	Index(ctx context.Context, audit *ExecutionAudit) error
	BatchIndex(ctx context.Context, audits []*ExecutionAudit) error
	Search(ctx context.Context, query string, from, size int) ([]*ExecutionAudit, int64, error)
}

// Read repositories

type MetricReadRepository interface {
	Save(ctx context.Context, m *Metric) error
	ListRecent(ctx context.Context, name string, limit int) ([]*Metric, error)
}

type SystemHealthReadRepository interface {
	Save(ctx context.Context, h *SystemHealth) error
	ListLatest(ctx context.Context, serviceName string, limit int) ([]*SystemHealth, error)
}

type AlertReadRepository interface {
	Save(ctx context.Context, a *Alert) error
	ListLatest(ctx context.Context, limit int) ([]*Alert, error)
}
