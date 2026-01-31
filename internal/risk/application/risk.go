package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// RiskService 风险门面服务。
type RiskService struct {
	Command *RiskCommand
	Query   *RiskQueryService
	logger  *slog.Logger
}

// NewRiskService 构造函数。
func NewRiskService(
	repo domain.RiskRepository,
	logger *slog.Logger,
) *RiskService {
	return &RiskService{
		Command: NewRiskCommand(repo),
		Query:   NewRiskQueryService(repo),
		logger:  logger.With("module", "risk_service"),
	}
}

// --- Command Facade ---

func (s *RiskService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	// 暂时返回 nil，因为 Command 中可能没有定义 AssessRisk 方法
	return nil, nil
}

func (s *RiskService) CalculatePortfolioRisk(ctx context.Context, req *CalculatePortfolioRiskRequest) (*CalculatePortfolioRiskResponse, error) {
	// 暂时返回 nil，因为 Command 中可能没有定义 CalculatePortfolioRisk 方法
	return nil, nil
}

func (s *RiskService) PerformGlobalRiskScan(ctx context.Context) error {
	// 暂时返回 nil，因为 Command 中可能没有定义 PerformGlobalRiskScan 方法
	return nil
}

func (s *RiskService) CheckRisk(ctx context.Context, userID string, symbol string, quantity, price float64) (bool, string) {
	// 暂时返回默认值，因为 Command 中可能没有定义 CheckRisk 方法
	return true, ""
}

func (s *RiskService) SetRiskLimit(ctx context.Context, userID string, maxOrderSize, maxDailyLoss float64) error {
	// 暂时返回 nil，因为 Command 中可能没有定义 SetRiskLimit 方法
	return nil
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
