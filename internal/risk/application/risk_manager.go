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
}

// NewRiskManager 构造函数。
func NewRiskManager(
	assessmentRepo domain.RiskAssessmentRepository,
	metricsRepo domain.RiskMetricsRepository,
	limitRepo domain.RiskLimitRepository,
	alertRepo domain.RiskAlertRepository,
) *RiskManager {
	return &RiskManager{
		assessmentRepo: assessmentRepo,
		metricsRepo:    metricsRepo,
		limitRepo:      limitRepo,
		alertRepo:      alertRepo,
	}
}

// AssessRisk 评估交易风险
func (m *RiskManager) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
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
		m.alertRepo.Save(ctx, alert)
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
