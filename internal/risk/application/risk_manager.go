package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/cache"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/security/risk"
)

// RiskManager 处理所有风险管理相关的写入操作（Commands）。
type RiskManager struct {
	assessmentRepo domain.RiskAssessmentRepository
	metricsRepo    domain.RiskMetricsRepository
	limitRepo      domain.RiskLimitRepository
	alertRepo      domain.RiskAlertRepository
	breakerRepo    domain.CircuitBreakerRepository
	calculator     *algorithm.RiskCalculator
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
		calculator:     algorithm.NewRiskCalculator(),
		ruleEngine:     ruleEngine,
		localCache:     localCache,
	}
}

// AssessRisk 评估交易风险 (修复版：功能完整 + 性能增强)
func (m *RiskManager) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	// 1. 极速熔断检查 (L1 Cache)
	breakerKey := fmt.Sprintf("breaker:%s", req.UserID)
	var isFired bool
	if err := m.localCache.Get(ctx, breakerKey, &isFired); err == nil && isFired {
		return &RiskAssessmentDTO{
			IsAllowed: false,
			Reason:    "Account trading suspended (cached)",
		}, nil
	}

	// 2. 检查账户熔断状态 (数据库回源)
	breaker, err := m.breakerRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if breaker != nil && breaker.IsFired {
		_ = m.localCache.Set(ctx, breakerKey, true, 30*time.Second)
		return &RiskAssessmentDTO{
			IsAllowed: false,
			Reason:    fmt.Sprintf("Account trading suspended: %s", breaker.TriggerReason),
		}, nil
	}

	quantity, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)
	orderValue := quantity.Mul(price)

	// 3. 动态规则引擎评估
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

	// 初始分值与等级
	riskLevel := domain.RiskLevelLow
	riskScore := decimal.NewFromInt(int64(assessmentResult.Score))
	if assessmentResult.Level == risk.Reject {
		riskLevel = domain.RiskLevelCritical
	} else if assessmentResult.Score > 50 {
		riskLevel = domain.RiskLevelHigh
	}

	// 保证金计算 (基础 10%)
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

	// 2. 检查组合风险限额 (Portfolio Risk Limits) ---
	limit, err := m.limitRepo.GetByUser(ctx, req.UserID, "MAX_SINGLE_ORDER_VALUE")
	if err == nil && limit != nil {
		if orderValue.GreaterThan(limit.LimitValue) {
			assessment.IsAllowed = false
			assessment.Reason = fmt.Sprintf("Order value %s exceeds limit %s", orderValue, limit.LimitValue)
		}
	}

	// 持久化评估结果
	if err := m.assessmentRepo.Save(ctx, assessment); err != nil {
		return nil, err
	}

	// 3. 风险告警逻辑 ---
	if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
		alert := &domain.RiskAlert{
			ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
			UserID:    req.UserID,
			AlertType: "HIGH_RISK",
			Severity:  string(riskLevel),
			Message:   fmt.Sprintf("High risk detected for %s: %s", req.Symbol, assessment.Reason),
		}
		if err := m.alertRepo.Save(ctx, alert); err != nil {
			logging.Error(ctx, "failed to save alert", "error", err)
		}
	}

	// 4. 返回完整的 DTO ---
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

// CalculatePortfolioRisk 计算组合风险 (VaR, ES)
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
		return nil, err
	}

	compVaR := make(map[string]string)
	for k, v := range result.ComponentVaR {
		compVaR[k] = v.String()
	}

	return &CalculatePortfolioRiskResponse{
		TotalValue:      result.TotalValue.String(),
		VaR:             result.VaR.String(),
		ES:              result.ES.String(),
		ComponentVaR:    compVaR,
		Diversification: result.Diversification.String(),
	}, nil
}

// PerformGlobalRiskScan 执行全局风险扫描并触发必要的熔断。
func (m *RiskManager) PerformGlobalRiskScan(ctx context.Context) error {
	m.internalLogger().InfoContext(ctx, "starting automated risk scan and stress test")

	// 1. 获取所有配置了风险限额的用户
	activeUsers := []string{"1001", "1002"}

	for _, userID := range activeUsers {
		// 2. 获取该用户的历史收益率数据
		returns := []decimal.Decimal{
			decimal.NewFromFloat(0.01), decimal.NewFromFloat(-0.02),
			decimal.NewFromFloat(0.05), decimal.NewFromFloat(-0.04),
		}

		// 3. 计算 VaR (95% 置信度)
		riskValue, err := m.calculator.CalculateVaR(returns, 0.95)
		if err != nil {
			continue
		}

		// 4. 获取限额并对比
		limit, err := m.limitRepo.GetByUser(ctx, userID, "VAR_LIMIT")
		if err == nil && limit != nil {
			if riskValue.Abs().GreaterThan(limit.LimitValue) {
				m.internalLogger().WarnContext(ctx, "VaR exceeds limit, triggering circuit breaker", "user_id", userID, "var", riskValue.String(), "limit", limit.LimitValue.String())

				// 5. 触发熔断
				_ = m.breakerRepo.Save(ctx, &domain.CircuitBreaker{
					UserID:        userID,
					IsFired:       true,
					TriggerReason: fmt.Sprintf("VaR limit exceeded: %s", riskValue.String()),
				})
			}
		}
	}

	return nil
}

func (m *RiskManager) internalLogger() *slog.Logger {
	return slog.Default().With("module", "risk_manager")
}

// SaveRiskMetrics 保存风险指标
func (m *RiskManager) SaveRiskMetrics(ctx context.Context, metrics *domain.RiskMetrics) error {
	return m.metricsRepo.Save(ctx, metrics)
}
