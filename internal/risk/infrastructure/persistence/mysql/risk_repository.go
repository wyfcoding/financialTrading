package mysql

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"gorm.io/gorm"
)

// RiskAssessmentModel 风险评估数据库模型
type RiskAssessmentModel struct {
	gorm.Model
	ID                string `gorm:"column:id;type:varchar(36);primaryKey"`
	UserID            string `gorm:"column:user_id;type:varchar(36);index;not null"`
	Symbol            string `gorm:"column:symbol;type:varchar(20);not null"`
	Side              string `gorm:"column:side;type:varchar(10);not null"`
	Quantity          string `gorm:"column:quantity;type:decimal(20,8);not null"`
	Price             string `gorm:"column:price;type:decimal(20,8);not null"`
	RiskLevel         string `gorm:"column:risk_level;type:varchar(20);not null"`
	RiskScore         string `gorm:"column:risk_score;type:decimal(5,2);not null"`
	MarginRequirement string `gorm:"column:margin_requirement;type:decimal(20,8);not null"`
	IsAllowed         bool   `gorm:"column:is_allowed;type:boolean;not null"`
	Reason            string `gorm:"column:reason;type:text"`
}

func (RiskAssessmentModel) TableName() string {
	return "risk_assessments"
}

func (m *RiskAssessmentModel) ToDomain() *domain.RiskAssessment {
	q, _ := decimal.NewFromString(m.Quantity)
	p, _ := decimal.NewFromString(m.Price)
	rs, _ := decimal.NewFromString(m.RiskScore)
	mr, _ := decimal.NewFromString(m.MarginRequirement)
	return &domain.RiskAssessment{
		Model:             m.Model,
		ID:                m.ID,
		UserID:            m.UserID,
		Symbol:            m.Symbol,
		Side:              m.Side,
		Quantity:          q,
		Price:             p,
		RiskLevel:         domain.RiskLevel(m.RiskLevel),
		RiskScore:         rs,
		MarginRequirement: mr,
		IsAllowed:         m.IsAllowed,
		Reason:            m.Reason,
	}
}

func FromAssessmentDomain(d *domain.RiskAssessment) *RiskAssessmentModel {
	return &RiskAssessmentModel{
		Model:             d.Model,
		ID:                d.ID,
		UserID:            d.UserID,
		Symbol:            d.Symbol,
		Side:              d.Side,
		Quantity:          d.Quantity.String(),
		Price:             d.Price.String(),
		RiskLevel:         string(d.RiskLevel),
		RiskScore:         d.RiskScore.String(),
		MarginRequirement: d.MarginRequirement.String(),
		IsAllowed:         d.IsAllowed,
		Reason:            d.Reason,
	}
}

// RiskMetricsModel 风险指标数据库模型
type RiskMetricsModel struct {
	gorm.Model
	UserID      string `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null"`
	VaR95       string `gorm:"column:var_95;type:decimal(20,8);not null"`
	VaR99       string `gorm:"column:var_99;type:decimal(20,8);not null"`
	MaxDrawdown string `gorm:"column:max_drawdown;type:decimal(20,8);not null"`
	SharpeRatio string `gorm:"column:sharpe_ratio;type:decimal(20,8);not null"`
	Correlation string `gorm:"column:correlation;type:decimal(20,8);not null"`
}

func (RiskMetricsModel) TableName() string {
	return "risk_metrics"
}

func (m *RiskMetricsModel) ToDomain() *domain.RiskMetrics {
	v95, _ := decimal.NewFromString(m.VaR95)
	v99, _ := decimal.NewFromString(m.VaR99)
	md, _ := decimal.NewFromString(m.MaxDrawdown)
	sr, _ := decimal.NewFromString(m.SharpeRatio)
	corr, _ := decimal.NewFromString(m.Correlation)
	return &domain.RiskMetrics{
		Model:       m.Model,
		UserID:      m.UserID,
		VaR95:       v95,
		VaR99:       v99,
		MaxDrawdown: md,
		SharpeRatio: sr,
		Correlation: corr,
	}
}

func FromMetricsDomain(d *domain.RiskMetrics) *RiskMetricsModel {
	return &RiskMetricsModel{
		Model:       d.Model,
		UserID:      d.UserID,
		VaR95:       d.VaR95.String(),
		VaR99:       d.VaR99.String(),
		MaxDrawdown: d.MaxDrawdown.String(),
		SharpeRatio: d.SharpeRatio.String(),
		Correlation: d.Correlation.String(),
	}
}

// RiskLimitModel 风险限额数据库模型
type RiskLimitModel struct {
	gorm.Model
	ID           string `gorm:"column:id;type:varchar(36);primaryKey"`
	UserID       string `gorm:"column:user_id;type:varchar(36);index;not null"`
	LimitType    string `gorm:"column:limit_type;type:varchar(50);not null"`
	LimitValue   string `gorm:"column:limit_value;type:decimal(20,8);not null"`
	CurrentValue string `gorm:"column:current_value;type:decimal(20,8);not null"`
	IsExceeded   bool   `gorm:"column:is_exceeded;type:boolean;not null"`
}

func (RiskLimitModel) TableName() string {
	return "risk_limits"
}

func (m *RiskLimitModel) ToDomain() *domain.RiskLimit {
	lv, _ := decimal.NewFromString(m.LimitValue)
	cv, _ := decimal.NewFromString(m.CurrentValue)
	return &domain.RiskLimit{
		Model:        m.Model,
		ID:           m.ID,
		UserID:       m.UserID,
		LimitType:    m.LimitType,
		LimitValue:   lv,
		CurrentValue: cv,
		IsExceeded:   m.IsExceeded,
	}
}

func FromLimitDomain(d *domain.RiskLimit) *RiskLimitModel {
	return &RiskLimitModel{
		Model:        d.Model,
		ID:           d.ID,
		UserID:       d.UserID,
		LimitType:    d.LimitType,
		LimitValue:   d.LimitValue.String(),
		CurrentValue: d.CurrentValue.String(),
		IsExceeded:   d.IsExceeded,
	}
}

// RiskAlertModel 风险告警数据库模型
type RiskAlertModel struct {
	gorm.Model
	ID        string `gorm:"column:id;type:varchar(36);primaryKey"`
	UserID    string `gorm:"column:user_id;type:varchar(36);index;not null"`
	AlertType string `gorm:"column:alert_type;type:varchar(50);not null"`
	Severity  string `gorm:"column:severity;type:varchar(20);not null"`
	Message   string `gorm:"column:message;type:text;not null"`
}

func (RiskAlertModel) TableName() string {
	return "risk_alerts"
}

func (m *RiskAlertModel) ToDomain() *domain.RiskAlert {
	return &domain.RiskAlert{
		Model:     m.Model,
		ID:        m.ID,
		UserID:    m.UserID,
		AlertType: m.AlertType,
		Severity:  m.Severity,
		Message:   m.Message,
	}
}

func FromAlertDomain(d *domain.RiskAlert) *RiskAlertModel {
	return &RiskAlertModel{
		Model:     d.Model,
		ID:        d.ID,
		UserID:    d.UserID,
		AlertType: d.AlertType,
		Severity:  d.Severity,
		Message:   d.Message,
	}
}

type assessmentRepository struct {
	db *gorm.DB
}

func NewRiskAssessmentRepository(db *gorm.DB) domain.RiskAssessmentRepository {
	return &assessmentRepository{db: db}
}

func (r *assessmentRepository) Save(ctx context.Context, assessment *domain.RiskAssessment) error {
	model := FromAssessmentDomain(assessment)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	assessment.Model = model.Model
	return nil
}

func (r *assessmentRepository) Get(ctx context.Context, id string) (*domain.RiskAssessment, error) {
	var model RiskAssessmentModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *assessmentRepository) GetLatestByUser(ctx context.Context, userID string) (*domain.RiskAssessment, error) {
	var model RiskAssessmentModel
	if err := r.db.WithContext(ctx).Order("created_at desc").First(&model, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

type metricsRepository struct {
	db *gorm.DB
}

func NewRiskMetricsRepository(db *gorm.DB) domain.RiskMetricsRepository {
	return &metricsRepository{db: db}
}

func (r *metricsRepository) Save(ctx context.Context, metrics *domain.RiskMetrics) error {
	model := FromMetricsDomain(metrics)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	metrics.Model = model.Model
	return nil
}

func (r *metricsRepository) Get(ctx context.Context, userID string) (*domain.RiskMetrics, error) {
	var model RiskMetricsModel
	if err := r.db.WithContext(ctx).First(&model, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

type limitRepository struct {
	db *gorm.DB
}

func NewRiskLimitRepository(db *gorm.DB) domain.RiskLimitRepository {
	return &limitRepository{db: db}
}

func (r *limitRepository) Save(ctx context.Context, limit *domain.RiskLimit) error {
	model := FromLimitDomain(limit)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	limit.Model = model.Model
	return nil
}

func (r *limitRepository) Get(ctx context.Context, id string) (*domain.RiskLimit, error) {
	var model RiskLimitModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *limitRepository) GetByUser(ctx context.Context, userID string, limitType string) (*domain.RiskLimit, error) {
	var model RiskLimitModel
	if err := r.db.WithContext(ctx).First(&model, "user_id = ? AND limit_type = ?", userID, limitType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

type alertRepository struct {
	db *gorm.DB
}

func NewRiskAlertRepository(db *gorm.DB) domain.RiskAlertRepository {
	return &alertRepository{db: db}
}

func (r *alertRepository) Save(ctx context.Context, alert *domain.RiskAlert) error {
	model := FromAlertDomain(alert)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	alert.Model = model.Model
	return nil
}

func (r *alertRepository) GetByUser(ctx context.Context, userID string, limit int) ([]*domain.RiskAlert, error) {
	var models []RiskAlertModel
	if err := r.db.WithContext(ctx).Limit(limit).Order("created_at desc").Find(&models, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.RiskAlert, len(models))
	for i, m := range models {
		result[i] = m.ToDomain()
	}
	return result, nil
}

func (r *alertRepository) DeleteByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&RiskAlertModel{}, "id = ?", id).Error
}
