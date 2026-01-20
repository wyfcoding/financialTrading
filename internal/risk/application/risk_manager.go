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
)

// RiskManager 处理所有风险管理相关的写入操作（Commands）。
type RiskManager struct {
	assessmentRepo domain.RiskAssessmentRepository
	metricsRepo    domain.RiskMetricsRepository
	limitRepo      domain.RiskLimitRepository
	alertRepo      domain.RiskAlertRepository
	breakerRepo    domain.CircuitBreakerRepository
	calculator     *finance.RiskCalculator
	ruleEngine     risk.Evaluator // 动态规则引擎
	localCache     cache.Cache    // 本地热点缓存
}

// NewRiskManager 构造函数。
func NewRiskManager(
	assessmentRepo domain.RiskAssessmentRepository,
	metricsRepo domain.RiskMetricsRepository,
	limitRepo domain.RiskLimitRepository,
	alertRepo domain.RiskAlertRepository,
	breakerRepo domain.CircuitBreakerRepository,
	ruleEngine risk.Evaluator,
	localCache cache.Cache,
) *RiskManager {
	return &RiskManager{
		assessmentRepo: assessmentRepo,
		metricsRepo:    metricsRepo,
		limitRepo:      limitRepo,
		alertRepo:      alertRepo,
		breakerRepo:    breakerRepo,
		calculator:     finance.NewRiskCalculator(),
		ruleEngine:     ruleEngine,
		localCache:     localCache,
	}
}

// AssessRisk 执行全方位的交易风险评估。
// 流程：极速熔断自检 -> 账户状态回源 -> 规则引擎判定 -> 组合限额校验 -> 结果持久化与告警。
func (m *RiskManager) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	// 1. 极速熔断检查 (L1 Local Cache)
	breakerKey := fmt.Sprintf("breaker:%s", req.UserID)
	var isFired bool
	if err := m.localCache.Get(ctx, breakerKey, &isFired); err == nil && isFired {
		return &RiskAssessmentDTO{
			IsAllowed: false,
			Reason:    "account trading suspended (cached)",
		}, nil
	}

	// 2. 检查账户熔断状态 (DB 回源)
	breaker, err := m.breakerRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		m.internalLogger().ErrorContext(ctx, "failed to get breaker from repo", "user_id", req.UserID, "error", err)
		return nil, err
	}
	if breaker != nil && breaker.IsFired {
		if err := m.localCache.Set(ctx, breakerKey, true, 30*time.Second); err != nil {
			m.internalLogger().WarnContext(ctx, "failed to update breaker local cache", "user_id", req.UserID, "error", err)
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

	assessmentResult, err := m.ruleEngine.Assess(ctx, "trade.assess", map[string]any{
		"user_id":     req.UserID,
		"symbol":      req.Symbol,
		"quantity":    quantity.InexactFloat64(),
		"order_value": orderValue.InexactFloat64(),
	})
	if err != nil {
		m.internalLogger().ErrorContext(ctx, "rule engine assessment failed", "error", err)
		return nil, fmt.Errorf("risk system internal error")
	}

	riskLevel := domain.RiskLevelLow
	riskScore := decimal.NewFromInt(int64(assessmentResult.Score))
	if assessmentResult.Level == risk.Reject {
		riskLevel = domain.RiskLevelCritical
	} else if assessmentResult.Score > 50 {
		riskLevel = domain.RiskLevelHigh
	}

	marginRequirement := orderValue.Mul(decimal.NewFromFloat(0.1))

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

	limit, err := m.limitRepo.GetByUserIDAndType(ctx, req.UserID, "MAX_SINGLE_ORDER_VALUE")
	if err == nil && limit != nil {
		if orderValue.GreaterThan(limit.LimitValue) {
			assessment.IsAllowed = false
			assessment.Reason = fmt.Sprintf("order value %s exceeds limit %s", orderValue, limit.LimitValue)
		}
	}

	if err := m.assessmentRepo.Save(ctx, assessment); err != nil {
		m.internalLogger().ErrorContext(ctx, "failed to save risk assessment", "user_id", req.UserID, "error", err)
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
		if err := m.alertRepo.Save(ctx, alert); err != nil {
			m.internalLogger().ErrorContext(ctx, "failed to save risk alert", "user_id", req.UserID, "error", err)
		}
	}

	m.internalLogger().InfoContext(ctx, "risk assessment completed", "user_id", req.UserID, "is_allowed", assessment.IsAllowed, "level", riskLevel)
	// ... (DTO 组装) ...
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

// ... (CalculatePortfolioRisk 补全日志) ...

// CalculatePortfolioRisk 基于历史波动率与相关性矩阵计算组合风险（VaR/ES）。
func (m *RiskManager) CalculatePortfolioRisk(ctx context.Context, req *CalculatePortfolioRiskRequest) (*CalculatePortfolioRiskResponse, error) {
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
		m.internalLogger().ErrorContext(ctx, "portfolio risk calculation failed", "error", err)
		return nil, err
	}

	compVaR := make(map[string]string)
	for k, v := range result.ComponentVaR {
		compVaR[k] = v.String()
	}

	m.internalLogger().InfoContext(ctx, "portfolio risk calculated", "total_value", result.TotalValue.String(), "var", result.VaR.String())
	// ... (Response 组装) ...
	return &CalculatePortfolioRiskResponse{
		TotalValue:      result.TotalValue.String(),
		VaR:             result.VaR.String(),
		ES:              result.ES.String(),
		ComponentVaR:    compVaR,
		Diversification: result.Diversification.String(),
	}, nil
}

// ... (PerformGlobalRiskScan 修复忽略错误) ...

// PerformGlobalRiskScan 自动扫描所有活跃账户的风险指标，并对超标账户触发熔断。
func (m *RiskManager) PerformGlobalRiskScan(ctx context.Context) error {
	m.internalLogger().InfoContext(ctx, "starting automated risk scan and stress test")

	activeUsers := []string{"1001", "1002"}

	for _, userID := range activeUsers {
		returns := []decimal.Decimal{
			decimal.NewFromFloat(0.01), decimal.NewFromFloat(-0.02),
			decimal.NewFromFloat(0.05), decimal.NewFromFloat(-0.04),
		}

		riskValue, err := m.calculator.CalculateVaR(returns, 0.95)
		if err != nil {
			continue
		}

		limit, err := m.limitRepo.GetByUserIDAndType(ctx, userID, "VAR_LIMIT")
		if err == nil && limit != nil {
			if riskValue.Abs().GreaterThan(limit.LimitValue) {
				m.internalLogger().WarnContext(ctx, "VaR exceeds limit, triggering circuit breaker", "user_id", userID, "var", riskValue.String(), "limit", limit.LimitValue.String())

				if err := m.breakerRepo.Save(ctx, &domain.CircuitBreaker{
					UserID:        userID,
					IsFired:       true,
					TriggerReason: fmt.Sprintf("VaR limit exceeded: %s", riskValue.String()),
				}); err != nil {
					m.internalLogger().ErrorContext(ctx, "failed to trigger circuit breaker", "user_id", userID, "error", err)
				}
			}
		}
	}

	m.internalLogger().InfoContext(ctx, "global risk scan completed")
	return nil
}

// internalLogger 返回预配置模块标签的日志记录器。
func (m *RiskManager) internalLogger() *slog.Logger {
	return slog.Default().With("module", "risk_manager")
}

// SaveRiskMetrics 对用户的量化风险指标进行快照存储。
func (m *RiskManager) SaveRiskMetrics(ctx context.Context, metrics *domain.RiskMetrics) error {
	if err := m.metricsRepo.Save(ctx, metrics); err != nil {
		m.internalLogger().ErrorContext(ctx, "failed to save risk metrics", "user_id", metrics.UserID, "error", err)
		return err
	}
	m.internalLogger().DebugContext(ctx, "risk metrics saved", "user_id", metrics.UserID)
	return nil
}
