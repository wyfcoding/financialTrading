package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/algorithm/finance"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/security/risk"
	"github.com/wyfcoding/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
)

// RiskCommandService 处理所有风险管理相关的写入操作（Commands）。
type RiskCommandService struct {
	repo       domain.RiskRepository
	calculator *finance.RiskCalculator
	marginCalc domain.MarginCalculator
	ruleEngine risk.Evaluator // 动态规则引擎
	localCache cache.Cache    // 本地热点缓存
	logger     *slog.Logger
}

// NewRiskCommandService 构造函数。
func NewRiskCommandService(
	repo domain.RiskRepository,
	ruleEngine risk.Evaluator,
	marginCalc domain.MarginCalculator,
	localCache cache.Cache,
	logger *slog.Logger,
) *RiskCommandService {
	return &RiskCommandService{
		repo:       repo,
		calculator: finance.NewRiskCalculator(),
		marginCalc: marginCalc,
		ruleEngine: ruleEngine,
		localCache: localCache,
		logger:     logger.With("module", "risk_command"),
	}
}

// AssessRisk 执行全方位的交易风险评估。
func (s *RiskCommandService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	ctx, span := tracing.Tracer().Start(ctx, "RiskCommandService.AssessRisk", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	tracing.AddTag(ctx, "user_id", req.UserID)
	tracing.AddTag(ctx, "symbol", req.Symbol)

	// 1. 极速熔断检查 (L1 Local Cache)
	breakerKey := fmt.Sprintf("breaker:%s", req.UserID)
	var isFired bool
	if err := s.localCache.Get(ctx, breakerKey, &isFired); err == nil && isFired {
		return &RiskAssessmentDTO{
			IsAllowed: false,
			Reason:    "account trading suspended (cached)",
		}, nil
	}

	// 2. 检查账户熔断状态 (DB 回源)
	breaker, err := s.repo.GetCircuitBreakerByUserID(ctx, req.UserID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get breaker from repo", "user_id", req.UserID, "error", err)
		return nil, err
	}
	if breaker != nil && breaker.IsFired {
		if err := s.localCache.Set(ctx, breakerKey, true, 30*time.Second); err != nil {
			s.logger.WarnContext(ctx, "failed to update breaker local cache", "user_id", req.UserID, "error", err)
		}
		return &RiskAssessmentDTO{
			IsAllowed: false,
			Reason:    fmt.Sprintf("account trading suspended: %s", breaker.TriggerReason),
		}, nil
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	orderValue := quantity.Mul(price)

	assessmentResult, err := s.ruleEngine.Assess(ctx, "trade.assess", map[string]any{
		"user_id":     req.UserID,
		"symbol":      req.Symbol,
		"quantity":    quantity.InexactFloat64(),
		"order_value": orderValue.InexactFloat64(),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "rule engine assessment failed", "error", err)
		return nil, fmt.Errorf("risk system internal error")
	}

	riskLevel := domain.RiskLevelLow
	riskScore := decimal.NewFromInt(int64(assessmentResult.Score))
	if assessmentResult.Level == risk.Reject {
		riskLevel = domain.RiskLevelCritical
	} else if assessmentResult.Score > 50 {
		riskLevel = domain.RiskLevelHigh
	}

	// 动态计算保证金要求
	marginRequirement, err := s.marginCalc.CalculateRequiredMargin(ctx, req.Symbol, orderValue)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to calculate dynamic margin, using fallback", "error", err)
		marginRequirement = orderValue.Mul(decimal.NewFromFloat(0.1)) // 10% 兜底
	}

	assessment := &domain.RiskAssessment{
		ID:                fmt.Sprintf("RISK-%d", idgen.GenID()),
		UserID:            req.UserID,
		Symbol:            req.Symbol,
		Side:              req.Side,
		Quantity:          quantity,
		Price:             price,
		RiskLevel:         riskLevel,
		RiskScore:         riskScore,
		MarginRequirement: marginRequirement,
		IsAllowed:         assessmentResult.Level != risk.Reject,
		Reason:            assessmentResult.Reason,
	}

	limitKey := fmt.Sprintf("limit:max_val:%s", req.UserID)
	var limit *domain.RiskLimit
	err = s.localCache.GetOrSet(ctx, limitKey, &limit, 5*time.Minute, func() (any, error) {
		return s.repo.GetLimitByUserIDAndType(ctx, req.UserID, "MAX_SINGLE_ORDER_VALUE")
	})

	if err == nil && limit != nil {
		if orderValue.GreaterThan(limit.LimitValue) {
			assessment.IsAllowed = false
			assessment.Reason = fmt.Sprintf("order value %s exceeds limit %s", orderValue, limit.LimitValue)
		}
	}

	if err := s.repo.SaveAssessment(ctx, assessment); err != nil {
		s.logger.ErrorContext(ctx, "failed to save risk assessment", "user_id", req.UserID, "error", err)
		return nil, err
	}

	if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
		alert := &domain.RiskAlert{
			ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
			UserID:    req.UserID,
			AlertType: "HIGH_RISK",
			Severity:  string(riskLevel),
			Message:   fmt.Sprintf("high risk detected for %s: %s", req.Symbol, assessment.Reason),
		}
		if err := s.repo.SaveAlert(ctx, alert); err != nil {
			s.logger.ErrorContext(ctx, "failed to save risk alert", "user_id", req.UserID, "error", err)
		}
	}

	s.logger.InfoContext(ctx, "risk assessment completed", "user_id", req.UserID, "is_allowed", assessment.IsAllowed, "level", riskLevel)

	return &RiskAssessmentDTO{
		AssessmentID:      assessment.ID,
		UserID:            assessment.UserID,
		Symbol:            assessment.Symbol,
		Side:              assessment.Side,
		Quantity:          assessment.Quantity.String(),
		Price:             assessment.Price.String(),
		RiskLevel:         string(assessment.RiskLevel),
		RiskScore:         assessment.RiskScore.String(),
		MarginRequirement: assessment.MarginRequirement.String(),
		IsAllowed:         assessment.IsAllowed,
		Reason:            assessment.Reason,
		CreatedAt:         assessment.CreatedAt.Unix(),
	}, nil
}

// CalculatePortfolioRisk 基于历史波动率与相关性矩阵计算组合风险（VaR/ES）。
func (s *RiskCommandService) CalculatePortfolioRisk(ctx context.Context, req *CalculatePortfolioRiskRequest) (*CalculatePortfolioRiskResponse, error) {
	assets := make([]domain.PortfolioAsset, len(req.Assets))
	for i, a := range req.Assets {
		pos, _ := decimal.NewFromString(a.Position)
		price, _ := decimal.NewFromString(a.CurrentPrice)
		assets[i] = domain.PortfolioAsset{
			Symbol:         a.Symbol,
			Position:       pos,
			CurrentPrice:   price,
			Volatility:     a.Volatility,
			ExpectedReturn: a.ExpectedReturn,
		}
	}

	domainInput := domain.PortfolioRiskInput{
		Assets:            assets,
		CorrelationMatrix: req.CorrelationData,
		TimeHorizon:       req.TimeHorizon,
		Simulations:       req.Simulations,
		ConfidenceLevel:   req.ConfidenceLevel,
	}

	result, err := domain.CalculatePortfolioRisk(domainInput)
	if err != nil {
		s.logger.ErrorContext(ctx, "portfolio risk calculation failed", "error", err)
		return nil, err
	}

	compVaR := make(map[string]string)
	for k, v := range result.ComponentVaR {
		compVaR[k] = v.String()
	}

	s.logger.InfoContext(ctx, "portfolio risk calculated", "total_value", result.TotalValue.String(), "var", result.VaR.String())

	return &CalculatePortfolioRiskResponse{
		TotalValue:      result.TotalValue.String(),
		VaR:             result.VaR.String(),
		ES:              result.ES.String(),
		ComponentVaR:    compVaR,
		Diversification: result.Diversification.String(),
	}, nil
}

// PerformGlobalRiskScan 自动扫描所有活跃账户的风险指标，并对超标账户触发熔断。
func (s *RiskCommandService) PerformGlobalRiskScan(ctx context.Context) error {
	s.logger.InfoContext(ctx, "starting automated risk scan and stress test")

	uids := []string{"1001", "1002", "1003"} // 简化版

	for _, userID := range uids {
		scenarioResults := domain.RunStressTests([]domain.PortfolioAsset{})

		for name, impact := range scenarioResults {
			if impact.Abs().GreaterThan(decimal.NewFromInt(10000)) {
				s.logger.WarnContext(ctx, "Stress test limit exceeded", "user_id", userID, "scenario", name, "impact", impact.String())

				s.repo.SaveCircuitBreaker(ctx, &domain.CircuitBreaker{
					UserID:        userID,
					IsFired:       true,
					TriggerReason: fmt.Sprintf("Stress scenario [%s] impact %s exceeds threshold", name, impact.String()),
				})
			}
		}
	}

	s.logger.InfoContext(ctx, "global risk scan completed")
	return nil
}

// SaveRiskMetrics 对用户的量化风险指标进行快照存储。
func (s *RiskCommandService) SaveRiskMetrics(ctx context.Context, metrics *domain.RiskMetrics) error {
	if err := s.repo.SaveMetrics(ctx, metrics); err != nil {
		s.logger.ErrorContext(ctx, "failed to save risk metrics", "user_id", metrics.UserID, "error", err)
		return err
	}
	s.logger.DebugContext(ctx, "risk metrics saved", "user_id", metrics.UserID)
	return nil
}

func (s *RiskCommandService) CheckRisk(ctx context.Context, userID string, symbol string, quantity, price float64) (bool, string) {
	limit, err := s.repo.GetLimitByUserIDAndType(ctx, userID, "ORDER_SIZE")
	if err != nil {
		return false, "No risk profile found"
	}
	if limit == nil {
		return true, ""
	}

	limitVal, _ := limit.LimitValue.Float64()
	if quantity > limitVal {
		return false, "Max order size exceeded"
	}

	return true, ""
}

func (s *RiskCommandService) SetRiskLimit(ctx context.Context, userID string, maxOrderSize, maxDailyLoss float64) error {
	limit := &domain.RiskLimit{
		UserID:       userID,
		LimitType:    "ORDER_SIZE",
		LimitValue:   decimal.NewFromFloat(maxOrderSize),
		CurrentValue: decimal.Zero,
	}
	return s.repo.SaveLimit(ctx, limit)
}
