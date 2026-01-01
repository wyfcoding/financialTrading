package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskQuery 处理所有风险管理相关的查询操作（Queries）。
type RiskQuery struct {
	assessmentRepo domain.RiskAssessmentRepository
	metricsRepo    domain.RiskMetricsRepository
	limitRepo      domain.RiskLimitRepository
	alertRepo      domain.RiskAlertRepository
}

// NewRiskQuery 构造函数。
func NewRiskQuery(
	assessmentRepo domain.RiskAssessmentRepository,
	metricsRepo domain.RiskMetricsRepository,
	limitRepo domain.RiskLimitRepository,
	alertRepo domain.RiskAlertRepository,
) *RiskQuery {
	return &RiskQuery{
		assessmentRepo: assessmentRepo,
		metricsRepo:    metricsRepo,
		limitRepo:      limitRepo,
		alertRepo:      alertRepo,
	}
}

// GetRiskMetrics 获取风险指标
func (q *RiskQuery) GetRiskMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	return q.metricsRepo.Get(ctx, userID)
}

// CheckRiskLimit 检查风险限额
func (q *RiskQuery) CheckRiskLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	return q.limitRepo.GetByUser(ctx, userID, limitType)
}

// GetRiskAlerts 获取风险告警
func (q *RiskQuery) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	return q.alertRepo.GetByUser(ctx, userID, limit)
}
