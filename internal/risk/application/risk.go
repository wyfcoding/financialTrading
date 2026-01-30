package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/security/risk"
)

// RiskService 风险门面服务。
type RiskService struct {
	Command *RiskCommandService
	Query   *RiskQueryService
	logger  *slog.Logger
}

// NewRiskService 构造函数。
func NewRiskService(
	repo domain.RiskRepository,
	ruleEngine risk.Evaluator,
	marginCalc domain.MarginCalculator,
	localCache cache.Cache,
	logger *slog.Logger,
) *RiskService {
	return &RiskService{
		Command: NewRiskCommandService(repo, ruleEngine, marginCalc, localCache, logger),
		Query:   NewRiskQueryService(repo),
		logger:  logger.With("module", "risk_service"),
	}
}

// --- Command Facade ---

func (s *RiskService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	return s.Command.AssessRisk(ctx, req)
}

func (s *RiskService) CalculatePortfolioRisk(ctx context.Context, req *CalculatePortfolioRiskRequest) (*CalculatePortfolioRiskResponse, error) {
	return s.Command.CalculatePortfolioRisk(ctx, req)
}

func (s *RiskService) PerformGlobalRiskScan(ctx context.Context) error {
	return s.Command.PerformGlobalRiskScan(ctx)
}

func (s *RiskService) CheckRisk(ctx context.Context, userID string, symbol string, quantity, price float64) (bool, string) {
	return s.Command.CheckRisk(ctx, userID, symbol, quantity, price)
}

func (s *RiskService) SetRiskLimit(ctx context.Context, userID string, maxOrderSize, maxDailyLoss float64) error {
	return s.Command.SetRiskLimit(ctx, userID, maxOrderSize, maxDailyLoss)
}

// --- Query Facade ---

func (s *RiskService) GetRiskMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	return s.Query.GetRiskMetrics(ctx, userID)
}

func (s *RiskService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	return s.Query.CheckRiskLimit(ctx, userID, limitType)
}

func (s *RiskService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	return s.Query.GetRiskAlerts(ctx, userID, limit)
}
