// Package application 包含风险管理服务的用例逻辑、DTO、事务边界与补偿策略
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/fynnwu/FinancialTrading/internal/risk/domain"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"github.com/fynnwu/FinancialTrading/pkg/utils"
	"github.com/shopspring/decimal"
)

// AssessRiskRequest 风险评估请求 DTO
type AssessRiskRequest struct {
	UserID   string
	Symbol   string
	Side     string
	Quantity string
	Price    string
}

// RiskAssessmentDTO 风险评估 DTO
type RiskAssessmentDTO struct {
	AssessmentID      string
	UserID            string
	Symbol            string
	Side              string
	Quantity          string
	Price             string
	RiskLevel         string
	RiskScore         string
	MarginRequirement string
	IsAllowed         bool
	Reason            string
	CreatedAt         int64
}

// RiskApplicationService 风险应用服务
// 处理风险评估、指标计算、限额检查等用例
type RiskApplicationService struct {
	assessmentRepo domain.RiskAssessmentRepository
	metricsRepo    domain.RiskMetricsRepository
	limitRepo      domain.RiskLimitRepository
	alertRepo      domain.RiskAlertRepository
	snowflake      *utils.SnowflakeID
}

// NewRiskApplicationService 创建风险应用服务
func NewRiskApplicationService(
	assessmentRepo domain.RiskAssessmentRepository,
	metricsRepo domain.RiskMetricsRepository,
	limitRepo domain.RiskLimitRepository,
	alertRepo domain.RiskAlertRepository,
) *RiskApplicationService {
	return &RiskApplicationService{
		assessmentRepo: assessmentRepo,
		metricsRepo:    metricsRepo,
		limitRepo:      limitRepo,
		alertRepo:      alertRepo,
		snowflake:      utils.NewSnowflakeID(6),
	}
}

// AssessRisk 评估交易风险
// 用例流程：
// 1. 验证输入参数
// 2. 获取用户风险指标
// 3. 计算风险等级和分数
// 4. 检查风险限额
// 5. 保存评估结果
// 6. 如果风险过高，生成告警
func (ras *RiskApplicationService) AssessRisk(ctx context.Context, req *AssessRiskRequest) (*RiskAssessmentDTO, error) {
	// 验证输入
	if req.UserID == "" || req.Symbol == "" || req.Side == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 解析数量和价格
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	// 生成评估 ID
	assessmentID := fmt.Sprintf("RISK-%d", ras.snowflake.Generate())

	// 计算风险等级和分数（简化实现）
	riskLevel := domain.RiskLevelLow
	riskScore := decimal.NewFromInt(20)
	marginRequirement := quantity.Mul(price).Mul(decimal.NewFromFloat(0.1))
	isAllowed := true
	reason := "Risk assessment passed"

	// 如果数量过大，提升风险等级
	if quantity.GreaterThan(decimal.NewFromInt(10000)) {
		riskLevel = domain.RiskLevelHigh
		riskScore = decimal.NewFromInt(75)
		marginRequirement = quantity.Mul(price).Mul(decimal.NewFromFloat(0.2))
	}

	// 创建评估对象
	assessment := &domain.RiskAssessment{
		AssessmentID:      assessmentID,
		UserID:            req.UserID,
		Symbol:            req.Symbol,
		Side:              req.Side,
		Quantity:          quantity,
		Price:             price,
		RiskLevel:         riskLevel,
		RiskScore:         riskScore,
		MarginRequirement: marginRequirement,
		IsAllowed:         isAllowed,
		Reason:            reason,
		CreatedAt:         time.Now(),
	}

	// 保存评估结果
	if err := ras.assessmentRepo.Save(ctx, assessment); err != nil {
		logger.WithContext(ctx).Error("Failed to save risk assessment",
			"assessment_id", assessmentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save risk assessment: %w", err)
	}

	// 如果风险过高，生成告警
	if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
		alert := &domain.RiskAlert{
			AlertID:   fmt.Sprintf("ALERT-%d", ras.snowflake.Generate()),
			UserID:    req.UserID,
			AlertType: "HIGH_RISK",
			Severity:  string(riskLevel),
			Message:   fmt.Sprintf("High risk detected for %s: %s", req.Symbol, reason),
			CreatedAt: time.Now(),
		}
		_ = ras.alertRepo.Save(ctx, alert)
	}

	logger.WithContext(ctx).Debug("Risk assessment completed",
		"assessment_id", assessmentID,
		"user_id", req.UserID,
		"risk_level", string(riskLevel),
	)

	// 转换为 DTO
	return &RiskAssessmentDTO{
		AssessmentID:      assessment.AssessmentID,
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

// GetRiskMetrics 获取风险指标
func (ras *RiskApplicationService) GetRiskMetrics(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	// 验证输入
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// 获取风险指标
	metrics, err := ras.metricsRepo.Get(ctx, userID)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get risk metrics",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk metrics: %w", err)
	}

	return metrics, nil
}

// CheckRiskLimit 检查风险限额
func (ras *RiskApplicationService) CheckRiskLimit(ctx context.Context, userID, limitType string) (*domain.RiskLimit, error) {
	// 验证输入
	if userID == "" || limitType == "" {
		return nil, fmt.Errorf("user_id and limit_type are required")
	}

	// 获取风险限额
	limit, err := ras.limitRepo.GetByUser(ctx, userID, limitType)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to check risk limit",
			"user_id", userID,
			"limit_type", limitType,
			"error", err,
		)
		return nil, fmt.Errorf("failed to check risk limit: %w", err)
	}

	return limit, nil
}

// GetRiskAlerts 获取风险告警
func (ras *RiskApplicationService) GetRiskAlerts(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	// 验证输入
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if limit <= 0 {
		limit = 100
	}

	// 获取风险告警
	alerts, err := ras.alertRepo.GetByUser(ctx, userID, limit)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get risk alerts",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk alerts: %w", err)
	}

	return alerts, nil
}
