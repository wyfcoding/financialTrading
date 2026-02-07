package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskProjectionService 将写模型投影到读模型（Redis/ES）。
type RiskProjectionService struct {
	repo       domain.RiskRepository
	readRepo   domain.RiskReadRepository
	searchRepo domain.RiskSearchRepository
	logger     *slog.Logger
}

func NewRiskProjectionService(
	repo domain.RiskRepository,
	readRepo domain.RiskReadRepository,
	searchRepo domain.RiskSearchRepository,
	logger *slog.Logger,
) *RiskProjectionService {
	return &RiskProjectionService{
		repo:       repo,
		readRepo:   readRepo,
		searchRepo: searchRepo,
		logger:     logger,
	}
}

func (s *RiskProjectionService) RefreshAssessment(ctx context.Context, assessmentID string, syncSearch bool) error {
	if assessmentID == "" {
		return nil
	}
	assessment, err := s.repo.GetAssessment(ctx, assessmentID)
	if err != nil || assessment == nil {
		return err
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexAssessment(ctx, assessment); err != nil {
			s.logger.WarnContext(ctx, "failed to index assessment", "error", err, "assessment_id", assessmentID)
			return err
		}
	}
	return nil
}

func (s *RiskProjectionService) RefreshAlert(ctx context.Context, alertID string, syncSearch bool) error {
	if alertID == "" {
		return nil
	}
	alert, err := s.repo.GetAlertByID(ctx, alertID)
	if err != nil || alert == nil {
		return err
	}
	if syncSearch && s.searchRepo != nil {
		if err := s.searchRepo.IndexAlert(ctx, alert); err != nil {
			s.logger.WarnContext(ctx, "failed to index alert", "error", err, "alert_id", alertID)
			return err
		}
	}
	return nil
}

func (s *RiskProjectionService) RefreshRiskLimit(ctx context.Context, userID, limitType string) error {
	if userID == "" || limitType == "" || s.readRepo == nil {
		return nil
	}
	limit, err := s.repo.GetLimitByUserIDAndType(ctx, userID, limitType)
	if err != nil || limit == nil {
		return err
	}
	if err := s.readRepo.SaveLimit(ctx, userID, limit); err != nil {
		s.logger.WarnContext(ctx, "failed to update risk limit cache", "error", err, "user_id", userID, "limit_type", limitType)
	}
	return nil
}

func (s *RiskProjectionService) RefreshRiskMetrics(ctx context.Context, userID string) error {
	if userID == "" || s.readRepo == nil {
		return nil
	}
	metrics, err := s.repo.GetMetrics(ctx, userID)
	if err != nil || metrics == nil {
		return err
	}
	if err := s.readRepo.SaveMetrics(ctx, userID, metrics); err != nil {
		s.logger.WarnContext(ctx, "failed to update risk metrics cache", "error", err, "user_id", userID)
	}
	return nil
}

func (s *RiskProjectionService) RefreshCircuitBreaker(ctx context.Context, userID string) error {
	if userID == "" || s.readRepo == nil {
		return nil
	}
	cb, err := s.repo.GetCircuitBreakerByUserID(ctx, userID)
	if err != nil || cb == nil {
		return err
	}
	if err := s.readRepo.SaveCircuitBreaker(ctx, userID, cb); err != nil {
		s.logger.WarnContext(ctx, "failed to update circuit breaker cache", "error", err, "user_id", userID)
	}
	return nil
}
