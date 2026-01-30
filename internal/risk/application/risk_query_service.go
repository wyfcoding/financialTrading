package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskQueryService 处理所有风险管理相关的查询操作（Queries）。
type RiskQueryService struct {
	repo domain.RiskRepository
}

// NewRiskQueryService 构造函数。
func NewRiskQueryService(repo domain.RiskRepository) *RiskQueryService {
	return &RiskQueryService{repo: repo}
}

// GetRiskMetrics 获取风险指标
func (s *RiskQueryService) GetRiskMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	return s.repo.GetMetrics(ctx, userID)
}

// CheckRiskLimit 检查风险限额
func (s *RiskQueryService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	return s.repo.GetLimitByUserIDAndType(ctx, userID, limitType)
}

// GetRiskAlerts 获取风险告警
func (s *RiskQueryService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	return s.repo.GetAlertsByUser(ctx, userID, limit)
}
