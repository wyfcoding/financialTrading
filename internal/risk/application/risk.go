package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/security/risk"
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

// CalculatePortfolioRiskRequest 组合风险计算请求 DTO
type CalculatePortfolioRiskRequest struct {
	Assets          []PortfolioAssetDTO `json:"assets"`
	CorrelationData [][]float64         `json:"correlation_data"` // 相关系数矩阵
	TimeHorizon     float64             `json:"time_horizon"`     // 时间跨度(年)
	Simulations     int                 `json:"simulations"`      // 模拟次数
	ConfidenceLevel float64             `json:"confidence_level"` // 置信度
}

type PortfolioAssetDTO struct {
	Symbol         string  `json:"symbol"`
	Position       string  `json:"position"`        // 持仓数量
	CurrentPrice   string  `json:"current_price"`   // 当前价格
	Volatility     float64 `json:"volatility"`      // 年化波动率
	ExpectedReturn float64 `json:"expected_return"` // 预期年化收益率
}

// CalculatePortfolioRiskResponse 组合风险计算响应 DTO
type CalculatePortfolioRiskResponse struct {
	TotalValue      string            `json:"total_value"`
	VaR             string            `json:"var"`
	ES              string            `json:"es"`
	ComponentVaR    map[string]string `json:"component_var"`
	Diversification string            `json:"diversification"`
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
	breakerRepo domain.CircuitBreakerRepository,
	ruleEngine risk.Evaluator,
	localCache cache.Cache,
) *RiskService {
	return &RiskService{
		manager: NewRiskManager(assessmentRepo, metricsRepo, limitRepo, alertRepo, breakerRepo, ruleEngine, localCache),
		query:   NewRiskQuery(assessmentRepo, metricsRepo, limitRepo, alertRepo),
	}
}

// --- Manager (Writes) ---

func (s *RiskService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	return s.manager.AssessRisk(ctx, req)
}

func (s *RiskService) CalculatePortfolioRisk(ctx context.Context, req *CalculatePortfolioRiskRequest) (*CalculatePortfolioRiskResponse, error) {
	return s.manager.CalculatePortfolioRisk(ctx, req)
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
