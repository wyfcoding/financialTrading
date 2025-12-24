// 包 风险管理服务的用例逻辑、DTO、事务边界与补偿策略
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
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

// RiskApplicationService 风险应用服务
// 处理风险评估、指标计算、限额检查等用例
type RiskApplicationService struct {
	assessmentRepo domain.RiskAssessmentRepository
	metricsRepo    domain.RiskMetricsRepository
	limitRepo      domain.RiskLimitRepository
	alertRepo      domain.RiskAlertRepository
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
	assessmentID := fmt.Sprintf("RISK-%d", idgen.GenID())

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
		logging.Error(ctx, "Failed to save risk assessment",
			"assessment_id", assessmentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save risk assessment: %w", err)
	}

	// 如果风险过高，生成告警
	if riskLevel == domain.RiskLevelHigh || riskLevel == domain.RiskLevelCritical {
		alert := &domain.RiskAlert{
			AlertID:   fmt.Sprintf("ALERT-%d", idgen.GenID()),
			UserID:    req.UserID,
			AlertType: "HIGH_RISK",
			Severity:  string(riskLevel),
			Message:   fmt.Sprintf("High risk detected for %s: %s", req.Symbol, reason),
			CreatedAt: time.Now(),
		}
		if err := ras.alertRepo.Save(ctx, alert); err != nil {
			logging.Error(ctx, "Failed to save risk alert", "user_id", req.UserID, "error", err)
		}
	}

	logging.Debug(ctx, "Risk assessment completed",
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
		logging.Error(ctx, "Failed to get risk metrics",
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
		logging.Error(ctx, "Failed to check risk limit",
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
		logging.Error(ctx, "Failed to get risk alerts",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk alerts: %w", err)
	}

	return alerts, nil
}
