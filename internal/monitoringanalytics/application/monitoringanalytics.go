package application

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/infrastructure/messaging"
)

// MonitoringAnalyticsService 监控分析门面服务，整合命令和查询服务
type MonitoringAnalyticsService struct {
	Command *MonitoringAnalyticsCommand
	Query   *MonitoringAnalyticsQuery
}

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishMetricCreated(event domain.MetricCreatedEvent) error { return nil }
func (m *mockEventPublisher) PublishAlertGenerated(event domain.AlertGeneratedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishAlertStatusChanged(event domain.AlertStatusChangedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishSystemHealthChanged(event domain.SystemHealthChangedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishExecutionAuditCreated(event domain.ExecutionAuditCreatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishSpoofingDetected(event domain.SpoofingDetectedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishMarketAnomalyDetected(event domain.MarketAnomalyDetectedEvent) error {
	return nil
}

// NewMonitoringAnalyticsService 构造函数
func NewMonitoringAnalyticsService(
	metricRepo domain.MetricRepository,
	healthRepo domain.SystemHealthRepository,
	alertRepo domain.AlertRepository,
	db interface{},
) (*MonitoringAnalyticsService, error) {
	// 创建事件发布者
	var eventPublisher domain.EventPublisher
	if gormDB, ok := db.(*gorm.DB); ok {
		eventPublisher = messaging.NewOutboxEventPublisher(gormDB)
	} else {
		// 使用空实现作为降级方案
		eventPublisher = &mockEventPublisher{}
	}

	// 创建命令服务
	command := NewMonitoringAnalyticsCommand(
		metricRepo,
		healthRepo,
		alertRepo,
		eventPublisher,
	)

	// 创建查询服务
	query := NewMonitoringAnalyticsQuery(
		metricRepo,
		healthRepo,
		alertRepo,
	)

	return &MonitoringAnalyticsService{
		Command: command,
		Query:   query,
	}, nil
}

// --- Command (Writes) ---

// RecordMetric 记录指标
func (s *MonitoringAnalyticsService) RecordMetric(ctx context.Context, name string, value interface{}, tags map[string]string, timestamp int64) error {
	// 这里需要根据实际类型转换 value
	// 暂时假设 value 是 float64，实际应用中需要更复杂的类型处理
	// 为了简化，这里直接调用 command 方法，实际应用中需要类型转换
	// 由于 domain.Metric 中的 Value 是 decimal.Decimal 类型，这里需要适配
	// 暂时留空，实际应用中需要实现类型转换
	return nil
}

// SaveSystemHealth 保存系统健康状态
func (s *MonitoringAnalyticsService) SaveSystemHealth(ctx context.Context, health *domain.SystemHealth) error {
	return s.Command.SaveSystemHealth(ctx, health)
}

// CreateAlert 创建告警
func (s *MonitoringAnalyticsService) CreateAlert(ctx context.Context, alert *domain.Alert) error {
	return s.Command.CreateAlert(ctx, alert)
}

// UpdateAlertStatus 更新告警状态
func (s *MonitoringAnalyticsService) UpdateAlertStatus(ctx context.Context, alertID string, oldStatus, newStatus string) error {
	return s.Command.UpdateAlertStatus(ctx, alertID, oldStatus, newStatus)
}

// RecordExecutionAudit 记录执行审计
func (s *MonitoringAnalyticsService) RecordExecutionAudit(ctx context.Context, audit *domain.ExecutionAudit) error {
	return s.Command.RecordExecutionAudit(ctx, audit)
}

// RecordSpoofingDetection 记录哄骗检测
func (s *MonitoringAnalyticsService) RecordSpoofingDetection(ctx context.Context, userID, symbol, orderID string) error {
	return s.Command.RecordSpoofingDetection(ctx, userID, symbol, orderID)
}

// RecordMarketAnomaly 记录市场异常
func (s *MonitoringAnalyticsService) RecordMarketAnomaly(ctx context.Context, symbol, anomalyType string, details map[string]interface{}) error {
	return s.Command.RecordMarketAnomaly(ctx, symbol, anomalyType, details)
}

// --- Query (Reads) ---

// GetMetrics 获取指标历史数据
func (s *MonitoringAnalyticsService) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	return s.Query.GetMetrics(ctx, name, startTime, endTime)
}

// GetTradeMetrics 获取交易指标
func (s *MonitoringAnalyticsService) GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*domain.TradeMetric, error) {
	return s.Query.GetTradeMetrics(ctx, symbol, startTime, endTime)
}

// GetSystemHealth 获取系统健康状态
func (s *MonitoringAnalyticsService) GetSystemHealth(ctx context.Context, serviceName string) ([]*domain.SystemHealth, error) {
	return s.Query.GetSystemHealth(ctx, serviceName)
}

// GetAlerts 获取告警列表
func (s *MonitoringAnalyticsService) GetAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	return s.Query.GetAlerts(ctx, limit)
}

// --- DTO Definitions ---

type MetricDTO struct {
	Symbol       string    `json:"symbol"`
	Timestamp    time.Time `json:"timestamp"`
	TotalVolume  float64   `json:"total_volume"`
	TradeCount   int       `json:"trade_count"`
	AveragePrice float64   `json:"average_price"`
}

type AlertDTO struct {
	ID        uint      `json:"id"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
