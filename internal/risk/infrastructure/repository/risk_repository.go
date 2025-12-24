// 包 仓储实现
package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// RiskAssessmentModel 风险评估数据库模型
// 对应数据库中的 risk_assessments 表
type RiskAssessmentModel struct {
	gorm.Model
	// 评估 ID
	AssessmentID string `gorm:"column:assessment_id;type:varchar(50);uniqueIndex;not null;comment:评估ID" json:"assessment_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null;comment:用户ID" json:"user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);not null;comment:交易对" json:"symbol"`
	// 买卖方向
	Side string `gorm:"column:side;type:varchar(10);not null;comment:买卖方向" json:"side"`
	// 数量
	Quantity string `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 价格
	Price string `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// 风险等级
	RiskLevel string `gorm:"column:risk_level;type:varchar(20);not null" json:"risk_level"`
	// 风险分数
	RiskScore string `gorm:"column:risk_score;type:decimal(5,2);not null" json:"risk_score"`
	// 保证金要求
	MarginRequirement string `gorm:"column:margin_requirement;type:decimal(20,8);not null" json:"margin_requirement"`
	// 是否允许
	IsAllowed bool `gorm:"column:is_allowed;type:boolean;not null" json:"is_allowed"`
	// 原因
	Reason string `gorm:"column:reason;type:text" json:"reason"`
}

// 指定表名
func (RiskAssessmentModel) TableName() string {
	return "risk_assessments"
}

// RiskAssessmentRepositoryImpl 风险评估仓储实现
type RiskAssessmentRepositoryImpl struct {
	db *gorm.DB
}

// NewRiskAssessmentRepository 创建风险评估仓储
func NewRiskAssessmentRepository(database *gorm.DB) domain.RiskAssessmentRepository {
	return &RiskAssessmentRepositoryImpl{
		db: database,
	}
}

// Save 保存风险评估
func (rar *RiskAssessmentRepositoryImpl) Save(ctx context.Context, assessment *domain.RiskAssessment) error {
	model := &RiskAssessmentModel{
		Model:             assessment.Model,
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
	}

	if err := rar.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save risk assessment",
			"assessment_id", assessment.AssessmentID,
			"error", err,
		)
		return fmt.Errorf("failed to save risk assessment: %w", err)
	}

	assessment.Model = model.Model
	return nil
}

// Get 获取风险评估
func (rar *RiskAssessmentRepositoryImpl) Get(ctx context.Context, assessmentID string) (*domain.RiskAssessment, error) {
	var model RiskAssessmentModel

	if err := rar.db.WithContext(ctx).Where("assessment_id = ?", assessmentID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get risk assessment",
			"assessment_id", assessmentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk assessment: %w", err)
	}

	return rar.modelToDomain(&model), nil
}

// GetLatestByUser 获取用户最新评估
func (rar *RiskAssessmentRepositoryImpl) GetLatestByUser(ctx context.Context, userID string) (*domain.RiskAssessment, error) {
	var model RiskAssessmentModel

	if err := rar.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get latest risk assessment",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get latest risk assessment: %w", err)
	}

	return rar.modelToDomain(&model), nil
}

// 将数据库模型转换为领域对象
func (rar *RiskAssessmentRepositoryImpl) modelToDomain(model *RiskAssessmentModel) *domain.RiskAssessment {
	quantity, _ := decimal.NewFromString(model.Quantity)
	price, _ := decimal.NewFromString(model.Price)
	riskScore, _ := decimal.NewFromString(model.RiskScore)
	marginRequirement, _ := decimal.NewFromString(model.MarginRequirement)

	return &domain.RiskAssessment{
		Model:             model.Model,
		AssessmentID:      model.AssessmentID,
		UserID:            model.UserID,
		Symbol:            model.Symbol,
		Side:              model.Side,
		Quantity:          quantity,
		Price:             price,
		RiskLevel:         domain.RiskLevel(model.RiskLevel),
		RiskScore:         riskScore,
		MarginRequirement: marginRequirement,
		IsAllowed:         model.IsAllowed,
		Reason:            model.Reason,
		CreatedAt:         model.CreatedAt,
	}
}

// RiskMetricsModel 风险指标数据库模型
type RiskMetricsModel struct {
	gorm.Model
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);uniqueIndex;not null" json:"user_id"`
	// 在 95% 置信水平下的风险价值 (Value at Risk)
	VaR95 string `gorm:"column:var_95;type:decimal(20,8);not null" json:"var_95"`
	// 在 99% 置信水平下的风险价值 (Value at Risk)
	VaR99 string `gorm:"column:var_99;type:decimal(20,8);not null" json:"var_99"`
	// 最大回撤
	MaxDrawdown string `gorm:"column:max_drawdown;type:decimal(20,8);not null" json:"max_drawdown"`
	// 夏普比率
	SharpeRatio string `gorm:"column:sharpe_ratio;type:decimal(20,8);not null" json:"sharpe_ratio"`
	// 相关系数
	Correlation string `gorm:"column:correlation;type:decimal(20,8);not null" json:"correlation"`
}

// 指定表名
func (RiskMetricsModel) TableName() string {
	return "risk_metrics"
}

// RiskMetricsRepositoryImpl 风险指标仓储实现
type RiskMetricsRepositoryImpl struct {
	db *gorm.DB
}

// NewRiskMetricsRepository 创建风险指标仓储
func NewRiskMetricsRepository(database *gorm.DB) domain.RiskMetricsRepository {
	return &RiskMetricsRepositoryImpl{
		db: database,
	}
}

// Save 保存风险指标
func (rmr *RiskMetricsRepositoryImpl) Save(ctx context.Context, metrics *domain.RiskMetrics) error {
	model := &RiskMetricsModel{
		Model:       metrics.Model,
		UserID:      metrics.UserID,
		VaR95:       metrics.VaR95.String(),
		VaR99:       metrics.VaR99.String(),
		MaxDrawdown: metrics.MaxDrawdown.String(),
		SharpeRatio: metrics.SharpeRatio.String(),
		Correlation: metrics.Correlation.String(),
	}

	if err := rmr.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save risk metrics",
			"user_id", metrics.UserID,
			"error", err,
		)
		return fmt.Errorf("failed to save risk metrics: %w", err)
	}

	metrics.Model = model.Model
	return nil
}

// Get 获取风险指标
func (rmr *RiskMetricsRepositoryImpl) Get(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	var model RiskMetricsModel

	if err := rmr.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get risk metrics",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk metrics: %w", err)
	}

	return rmr.modelToDomain(&model), nil
}

// Update 更新风险指标
func (rmr *RiskMetricsRepositoryImpl) Update(ctx context.Context, metrics *domain.RiskMetrics) error {
	if err := rmr.db.WithContext(ctx).Model(&RiskMetricsModel{}).Where("user_id = ?", metrics.UserID).Updates(map[string]any{
		"var_95":       metrics.VaR95.String(),
		"var_99":       metrics.VaR99.String(),
		"max_drawdown": metrics.MaxDrawdown.String(),
		"sharpe_ratio": metrics.SharpeRatio.String(),
		"correlation":  metrics.Correlation.String(),
	}).Error; err != nil {
		logging.Error(ctx, "Failed to update risk metrics",
			"user_id", metrics.UserID,
			"error", err,
		)
		return fmt.Errorf("failed to update risk metrics: %w", err)
	}

	return nil
}

// 将数据库模型转换为领域对象
func (rmr *RiskMetricsRepositoryImpl) modelToDomain(model *RiskMetricsModel) *domain.RiskMetrics {
	var95, _ := decimal.NewFromString(model.VaR95)
	var99, _ := decimal.NewFromString(model.VaR99)
	maxDrawdown, _ := decimal.NewFromString(model.MaxDrawdown)
	sharpeRatio, _ := decimal.NewFromString(model.SharpeRatio)
	correlation, _ := decimal.NewFromString(model.Correlation)

	return &domain.RiskMetrics{
		Model:       model.Model,
		UserID:      model.UserID,
		VaR95:       var95,
		VaR99:       var99,
		MaxDrawdown: maxDrawdown,
		SharpeRatio: sharpeRatio,
		Correlation: correlation,
		UpdatedAt:   model.UpdatedAt,
	}
}

// RiskLimitModel 风险限额数据库模型
type RiskLimitModel struct {
	gorm.Model
	// 限额 ID
	LimitID string `gorm:"column:limit_id;type:varchar(50);uniqueIndex;not null" json:"limit_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 限额类型
	LimitType string `gorm:"column:limit_type;type:varchar(50);not null" json:"limit_type"`
	// 限额值
	LimitValue string `gorm:"column:limit_value;type:decimal(20,8);not null" json:"limit_value"`
	// 当前值
	CurrentValue string `gorm:"column:current_value;type:decimal(20,8);not null" json:"current_value"`
	// 是否超限
	IsExceeded bool `gorm:"column:is_exceeded;type:boolean;not null" json:"is_exceeded"`
}

// 指定表名
func (RiskLimitModel) TableName() string {
	return "risk_limits"
}

// RiskLimitRepositoryImpl 风险限额仓储实现
type RiskLimitRepositoryImpl struct {
	db *gorm.DB
}

// NewRiskLimitRepository 创建风险限额仓储
func NewRiskLimitRepository(database *gorm.DB) domain.RiskLimitRepository {
	return &RiskLimitRepositoryImpl{
		db: database,
	}
}

// Save 保存风险限额
func (rlr *RiskLimitRepositoryImpl) Save(ctx context.Context, limit *domain.RiskLimit) error {
	model := &RiskLimitModel{
		Model:        limit.Model,
		LimitID:      limit.LimitID,
		UserID:       limit.UserID,
		LimitType:    limit.LimitType,
		LimitValue:   limit.LimitValue.String(),
		CurrentValue: limit.CurrentValue.String(),
		IsExceeded:   limit.IsExceeded,
	}

	if err := rlr.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save risk limit",
			"limit_id", limit.LimitID,
			"error", err,
		)
		return fmt.Errorf("failed to save risk limit: %w", err)
	}

	limit.Model = model.Model
	return nil
}

// Get 获取风险限额
func (rlr *RiskLimitRepositoryImpl) Get(ctx context.Context, limitID string) (*domain.RiskLimit, error) {
	var model RiskLimitModel

	if err := rlr.db.WithContext(ctx).Where("limit_id = ?", limitID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get risk limit",
			"limit_id", limitID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk limit: %w", err)
	}

	return rlr.modelToDomain(&model), nil
}

// GetByUser 获取用户限额
func (rlr *RiskLimitRepositoryImpl) GetByUser(ctx context.Context, userID string, limitType string) (*domain.RiskLimit, error) {
	var model RiskLimitModel

	if err := rlr.db.WithContext(ctx).Where("user_id = ? AND limit_type = ?", userID, limitType).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logging.Error(ctx, "Failed to get risk limit by user",
			"user_id", userID,
			"limit_type", limitType,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk limit by user: %w", err)
	}

	return rlr.modelToDomain(&model), nil
}

// Update 更新风险限额
func (rlr *RiskLimitRepositoryImpl) Update(ctx context.Context, limit *domain.RiskLimit) error {
	if err := rlr.db.WithContext(ctx).Model(&RiskLimitModel{}).Where("limit_id = ?", limit.LimitID).Updates(map[string]any{
		"limit_value":   limit.LimitValue.String(),
		"current_value": limit.CurrentValue.String(),
		"is_exceeded":   limit.IsExceeded,
	}).Error; err != nil {
		logging.Error(ctx, "Failed to update risk limit",
			"limit_id", limit.LimitID,
			"error", err,
		)
		return fmt.Errorf("failed to update risk limit: %w", err)
	}

	return nil
}

// 将数据库模型转换为领域对象
func (rlr *RiskLimitRepositoryImpl) modelToDomain(model *RiskLimitModel) *domain.RiskLimit {
	limitValue, _ := decimal.NewFromString(model.LimitValue)
	currentValue, _ := decimal.NewFromString(model.CurrentValue)

	return &domain.RiskLimit{
		Model:        model.Model,
		LimitID:      model.LimitID,
		UserID:       model.UserID,
		LimitType:    model.LimitType,
		LimitValue:   limitValue,
		CurrentValue: currentValue,
		IsExceeded:   model.IsExceeded,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

// RiskAlertModel 风险告警数据库模型
type RiskAlertModel struct {
	gorm.Model
	// 告警 ID
	AlertID string `gorm:"column:alert_id;type:varchar(50);uniqueIndex;not null" json:"alert_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 告警类型
	AlertType string `gorm:"column:alert_type;type:varchar(50);not null" json:"alert_type"`
	// 严重程度
	Severity string `gorm:"column:severity;type:varchar(20);not null" json:"severity"`
	// 消息
	Message string `gorm:"column:message;type:text;not null" json:"message"`
}

// 指定表名
func (RiskAlertModel) TableName() string {
	return "risk_alerts"
}

// RiskAlertRepositoryImpl 风险告警仓储实现
type RiskAlertRepositoryImpl struct {
	db *gorm.DB
}

// NewRiskAlertRepository 创建风险告警仓储
func NewRiskAlertRepository(database *gorm.DB) domain.RiskAlertRepository {
	return &RiskAlertRepositoryImpl{
		db: database,
	}
}

// Save 保存风险告警
func (rar *RiskAlertRepositoryImpl) Save(ctx context.Context, alert *domain.RiskAlert) error {
	model := &RiskAlertModel{
		Model:     alert.Model,
		AlertID:   alert.AlertID,
		UserID:    alert.UserID,
		AlertType: alert.AlertType,
		Severity:  alert.Severity,
		Message:   alert.Message,
	}

	if err := rar.db.WithContext(ctx).Create(model).Error; err != nil {
		logging.Error(ctx, "Failed to save risk alert",
			"alert_id", alert.AlertID,
			"error", err,
		)
		return fmt.Errorf("failed to save risk alert: %w", err)
	}

	alert.Model = model.Model
	return nil
}

// GetByUser 获取用户告警
func (rar *RiskAlertRepositoryImpl) GetByUser(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	var models []RiskAlertModel

	if err := rar.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&models).Error; err != nil {
		logging.Error(ctx, "Failed to get risk alerts by user",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get risk alerts by user: %w", err)
	}

	alerts := make([]*domain.RiskAlert, 0, len(models))
	for _, model := range models {
		alerts = append(alerts, rar.modelToDomain(&model))
	}

	return alerts, nil
}

// DeleteRead 删除已读告警
func (rar *RiskAlertRepositoryImpl) DeleteRead(ctx context.Context, alertID string) error {
	if err := rar.db.WithContext(ctx).Where("alert_id = ?", alertID).Delete(&RiskAlertModel{}).Error; err != nil {
		logging.Error(ctx, "Failed to delete risk alert",
			"alert_id", alertID,
			"error", err,
		)
		return fmt.Errorf("failed to delete risk alert: %w", err)
	}

	return nil
}

// 将数据库模型转换为领域对象
func (rar *RiskAlertRepositoryImpl) modelToDomain(model *RiskAlertModel) *domain.RiskAlert {
	return &domain.RiskAlert{
		Model:     model.Model,
		AlertID:   model.AlertID,
		UserID:    model.UserID,
		AlertType: model.AlertType,
		Severity:  model.Severity,
		Message:   model.Message,
		CreatedAt: model.CreatedAt,
	}
}
