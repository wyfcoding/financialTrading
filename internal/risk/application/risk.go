package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// AssessRiskRequest 风险评估请求 DTO
type AssessRiskRequest struct {
	UserID   string // 用户 ID
	Symbol   string // 交易对
	Side     string // 买卖方向
	Quantity string // 数量
	Price    string // 价格
}

// RiskAssessmentDTO 风险评估 DTO
type RiskAssessmentDTO struct {
	AssessmentID      string // 评估 ID
	UserID            string // 用户 ID
	Symbol            string // 交易对
	Side              string // 买卖方向
	Quantity          string // 数量
	Price             string // 价格
	RiskLevel         string // 风险等级
	RiskScore         string // 风险评分
	MarginRequirement string // 保证金要求
	IsAllowed         bool   // 是否允许交易
	Reason            string // 原因
	CreatedAt         int64  // 创建时间戳
}

// RiskService 风险门面服务，整合 Manager 和 Query。
type RiskService struct {
	manager *RiskManager
	query   *RiskQuery
}

// NewRiskService 构造函数。
func NewRiskService(
	assessmentRepo domain.RiskAssessmentRepository,
	metricsRepo domain.RiskMetricsRepository,
	limitRepo domain.RiskLimitRepository,
	alertRepo domain.RiskAlertRepository,
) *RiskService {
	return &RiskService{
		manager: NewRiskManager(assessmentRepo, metricsRepo, limitRepo, alertRepo),
		query:   NewRiskQuery(assessmentRepo, metricsRepo, limitRepo, alertRepo),
	}
}

// --- Manager (Writes) ---

func (s *RiskService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	return s.manager.AssessRisk(ctx, req)
}

func (s *RiskService) PerformGlobalRiskScan(ctx context.Context) error {
	return s.manager.PerformGlobalRiskScan(ctx)
}

// --- Query (Reads) ---

func (s *RiskService) GetRiskMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	return s.query.GetRiskMetrics(ctx, userID)
}

func (s *RiskService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	return s.query.CheckRiskLimit(ctx, userID, limitType)
}

func (s *RiskService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	return s.query.GetRiskAlerts(ctx, userID, limit)
}
