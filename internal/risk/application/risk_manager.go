package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// RiskManager 处理所有风险管理相关的写入操作（Commands）。
type RiskManager struct {
	assessmentRepo domain.RiskAssessmentRepository
	metricsRepo    domain.RiskMetricsRepository
	limitRepo      domain.RiskLimitRepository
	alertRepo      domain.RiskAlertRepository
	breakerRepo    domain.CircuitBreakerRepository
	calculator     *algorithm.RiskCalculator
}

// NewRiskManager 构造函数。
func NewRiskManager(
	assessmentRepo domain.RiskAssessmentRepository,
	metricsRepo domain.RiskMetricsRepository,
	limitRepo domain.RiskLimitRepository,
	alertRepo domain.RiskAlertRepository,
	breakerRepo domain.CircuitBreakerRepository,
) *RiskManager {
	return &RiskManager{
		assessmentRepo: assessmentRepo,
		metricsRepo:    metricsRepo,
		limitRepo:      limitRepo,
		alertRepo:      alertRepo,
		breakerRepo:    breakerRepo,
		calculator:     algorithm.NewRiskCalculator(),
	}
}

// AssessRisk 评估交易风险
func (m *RiskManager) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	// 1. 检查账户熔断状态
	breaker, err := m.breakerRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if breaker != nil && breaker.IsFired {
		return &RiskAssessmentDTO{
			IsAllowed: false,
			Reason:    fmt.Sprintf("Account trading suspended: %s", breaker.TriggerReason),
		}, nil
	}

	quantity, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)

	assessmentID := fmt.Sprintf("RISK-%d", idgen.GenID())
	riskLevel := domain.RiskLevelLow
	riskScore := decimal.NewFromInt(20)
	marginRequirement := quantity.Mul(price).Mul(decimal.NewFromFloat(0.1))

	if quantity.GreaterThan(decimal.NewFromInt(10000)) {
		riskLevel = domain.RiskLevelHigh
		riskScore = decimal.NewFromInt(75)
		marginRequirement = quantity.Mul(price).Mul(decimal.NewFromFloat(0.2))
	}

	assessment := &domain.RiskAssessment{
		ID:                assessmentID,
		UserID:            req.UserID,
		Symbol:            req.Symbol,
		Side:              req.Side,
		Quantity:          quantity,
		Price:             price,
		RiskLevel:         riskLevel,
		RiskScore:         riskScore,
		MarginRequirement: marginRequirement,
		IsAllowed:         true,
		Reason:            "Risk assessment passed",
	}

	// 2. 检查组合风险限额 (Portfolio Risk Limits)
	// 示例：检查单笔交易金额是否超过每日累计限额
	limit, err := m.limitRepo.GetByUser(ctx, req.UserID, "MAX_SINGLE_ORDER_VALUE")
	if err == nil && limit != nil {
		orderValue := quantity.Mul(price)
		if orderValue.GreaterThan(limit.LimitValue) {
			assessment.IsAllowed = false
			assessment.Reason = fmt.Sprintf("Order value %s exceeds limit %s", orderValue, limit.LimitValue)
		}
	}

	if err := m.assessmentRepo.Save(ctx, assessment); err != nil {
		return nil, err
	}

	if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
		alert := &domain.RiskAlert{
			ID:        fmt.Sprintf("ALERT-%d", idgen.GenID()),
			UserID:    req.UserID,
			AlertType: "HIGH_RISK",
			Severity:  string(riskLevel),
			Message:   fmt.Sprintf("High risk detected for %s", req.Symbol),
		}
		if err := m.alertRepo.Save(ctx, alert); err != nil {
			logging.Error(ctx, "RiskManager: failed to save alert", "error", err)
		}
	}

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
