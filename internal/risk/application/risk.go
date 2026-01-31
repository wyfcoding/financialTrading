package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
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
	redisRepo domain.RiskRedisRepository,
	logger *slog.Logger,
) *RiskService {
	return &RiskService{
		Command: NewRiskCommand(repo, redisRepo),
		Query:   NewRiskQueryService(repo, redisRepo),
		logger:  logger.With("module", "risk_service"),
	}
}

// --- Command Facade ---

func (s *RiskService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	qty, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)

	cmd := AssessRiskCommand{
		UserID:   req.UserID,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Quantity: qty.InexactFloat64(),
		Price:    price.InexactFloat64(),
	}

	agg, err := s.Command.AssessRisk(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &RiskAssessmentDTO{
		AssessmentID:      agg.ID,
		UserID:            agg.UserID,
		Symbol:            agg.Symbol,
		Side:              agg.Side,
		Quantity:          agg.Quantity.String(),
		Price:             agg.Price.String(),
		RiskLevel:         string(agg.RiskLevel),
		RiskScore:         agg.RiskScore.String(),
		MarginRequirement: agg.MarginRequirement.String(),
		IsAllowed:         agg.IsAllowed,
		Reason:            agg.Reason,
		CreatedAt:         agg.CreatedAt.Unix(),
	}, nil
}

func (s *RiskService) SetRiskLimit(ctx context.Context, userID, limitType string, value float64) error {
	cmd := UpdateRiskLimitCommand{
		UserID:     userID,
		LimitType:  limitType,
		LimitValue: value,
	}
	_, err := s.Command.UpdateRiskLimit(ctx, cmd)
	return err
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

// --- DTO Definitions ---

// AssessRiskRequest 风险评估请求 DTO
type AssessRiskRequest struct {
	UserID   string `json:"user_id"`
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Quantity string `json:"quantity"`
	Price    string `json:"price"`
}

// RiskAssessmentDTO 风险评估 DTO
type RiskAssessmentDTO struct {
	AssessmentID      string `json:"assessment_id"`
	UserID            string `json:"user_id"`
	Symbol            string `json:"symbol"`
	Side              string `json:"side"`
	Quantity          string `json:"quantity"`
	Price             string `json:"price"`
	RiskLevel         string `json:"risk_level"`
	RiskScore         string `json:"risk_score"`
	MarginRequirement string `json:"margin_requirement"`
	IsAllowed         bool   `json:"is_allowed"`
	Reason            string `json:"reason"`
	CreatedAt         int64  `json:"created_at"`
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
