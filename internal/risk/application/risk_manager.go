package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
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

// PerformGlobalRiskScan 执行全局风险扫描
func (m *RiskManager) PerformGlobalRiskScan(ctx context.Context) error {
	logging.Info(ctx, "Starting global risk metrics scan")
	return nil
}

// SaveRiskMetrics 保存风险指标
func (m *RiskManager) SaveRiskMetrics(ctx context.Context, metrics *domain.RiskMetrics) error {
	return m.metricsRepo.Save(ctx, metrics)
}
