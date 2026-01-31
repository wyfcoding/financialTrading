package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskQueryService 处理所有风险管理相关的查询操作（Queries）。
type RiskQueryService struct {
	repo      domain.RiskRepository
	redisRepo domain.RiskRedisRepository
}

// NewRiskQueryService 构造函数。
func NewRiskQueryService(repo domain.RiskRepository, redisRepo domain.RiskRedisRepository) *RiskQueryService {
	return &RiskQueryService{
		repo:      repo,
		redisRepo: redisRepo,
	}
}

// GetRiskMetrics 获取风险指标（优先缓存）
func (s *RiskQueryService) GetRiskMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	// 尝试从缓存获取
	if cached, err := s.redisRepo.GetMetrics(ctx, userID); err == nil && cached != nil {
		return cached, nil
	}

	// 从主库获取
	metrics, err := s.repo.GetMetrics(ctx, userID)
	if err == nil && metrics != nil {
		// 回填缓存
		_ = s.redisRepo.SaveMetrics(ctx, userID, metrics)
	}
	return metrics, err
}

// CheckRiskLimit 检查风险限额（优先缓存）
func (s *RiskQueryService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	if cached, err := s.redisRepo.GetLimit(ctx, userID, limitType); err == nil && cached != nil {
		return cached, nil
	}

	limit, err := s.repo.GetLimitByUserIDAndType(ctx, userID, limitType)
	if err == nil && limit != nil {
		_ = s.redisRepo.SaveLimit(ctx, userID, limit)
	}
	return limit, err
}

// GetRiskAlerts 获取风险告警（告警不设缓存，实时从库获取）
func (s *RiskQueryService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	return s.repo.GetAlertsByUser(ctx, userID, limit)
}

// GetCircuitBreaker 获取熔断状态（优先缓存）
func (s *RiskQueryService) GetCircuitBreaker(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
	if cached, err := s.redisRepo.GetCircuitBreaker(ctx, userID); err == nil && cached != nil {
		return cached, nil
	}

	cb, err := s.repo.GetCircuitBreakerByUserID(ctx, userID)
	if err == nil && cb != nil {
		_ = s.redisRepo.SaveCircuitBreaker(ctx, userID, cb)
	}
	return cb, err
}
