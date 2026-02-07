package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskQueryService 处理所有风险管理相关的查询操作（Queries）。
type RiskQueryService struct {
	repo       domain.RiskRepository
	readRepo   domain.RiskReadRepository
	searchRepo domain.RiskSearchRepository
}

// NewRiskQueryService 构造函数。
func NewRiskQueryService(repo domain.RiskRepository, readRepo domain.RiskReadRepository, searchRepo domain.RiskSearchRepository) *RiskQueryService {
	return &RiskQueryService{
		repo:       repo,
		readRepo:   readRepo,
		searchRepo: searchRepo,
	}
}

// GetRiskMetrics 获取风险指标（优先缓存）
func (s *RiskQueryService) GetRiskMetrics(ctx context.Context, userID string) (*RiskMetricsDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetMetrics(ctx, userID); err == nil && cached != nil {
			return toRiskMetricsDTO(cached), nil
		}
	}

	metrics, err := s.repo.GetMetrics(ctx, userID)
	if err != nil {
		return nil, err
	}
	if metrics != nil && s.readRepo != nil {
		_ = s.readRepo.SaveMetrics(ctx, userID, metrics)
	}
	return toRiskMetricsDTO(metrics), nil
}

// CheckRiskLimit 检查风险限额（优先缓存）
func (s *RiskQueryService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimitDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetLimit(ctx, userID, limitType); err == nil && cached != nil {
			return toRiskLimitDTO(cached), nil
		}
	}

	limit, err := s.repo.GetLimitByUserIDAndType(ctx, userID, limitType)
	if err != nil {
		return nil, err
	}
	if limit != nil && s.readRepo != nil {
		_ = s.readRepo.SaveLimit(ctx, userID, limit)
	}
	return toRiskLimitDTO(limit), nil
}

// GetRiskAlerts 获取风险告警（告警不设缓存）
func (s *RiskQueryService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*RiskAlertDTO, error) {
	alerts, err := s.repo.GetAlertsByUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return toRiskAlertDTOs(alerts), nil
}

// GetCircuitBreaker 获取熔断状态（优先缓存）
func (s *RiskQueryService) GetCircuitBreaker(ctx context.Context, userID string) (*CircuitBreakerDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetCircuitBreaker(ctx, userID); err == nil && cached != nil {
			return toCircuitBreakerDTO(cached), nil
		}
	}

	cb, err := s.repo.GetCircuitBreakerByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cb != nil && s.readRepo != nil {
		_ = s.readRepo.SaveCircuitBreaker(ctx, userID, cb)
	}
	return toCircuitBreakerDTO(cb), nil
}

// SearchRiskAssessments 搜索风险评估
func (s *RiskQueryService) SearchRiskAssessments(ctx context.Context, userID, symbol string, level domain.RiskLevel, limit, offset int) ([]*RiskAssessmentDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	assessments, total, err := s.searchRepo.SearchAssessments(ctx, userID, symbol, level, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*RiskAssessmentDTO, 0, len(assessments))
	for _, a := range assessments {
		result = append(result, toRiskAssessmentDTO(a))
	}
	return result, total, nil
}

// SearchRiskAlerts 搜索风险告警
func (s *RiskQueryService) SearchRiskAlerts(ctx context.Context, userID, severity, alertType string, limit, offset int) ([]*RiskAlertDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	alerts, total, err := s.searchRepo.SearchAlerts(ctx, userID, severity, alertType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toRiskAlertDTOs(alerts), total, nil
}
