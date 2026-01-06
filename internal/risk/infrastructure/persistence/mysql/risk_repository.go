// Package mysql 提供了风险管理模块各领域实体的 MySQL GORM 持久化实现。
package mysql

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	"gorm.io/gorm"
)

// RiskAssessmentModel 风险评估记录数据库模型。
type RiskAssessmentModel struct {
	gorm.Model
	ID                string `gorm:"column:id;type:varchar(36);primaryKey;comment:评估唯一ID"`
	UserID            string `gorm:"column:user_id;type:varchar(36);index;not null;comment:用户ID"`
	Symbol            string `gorm:"column:symbol;type:varchar(20);not null;comment:交易对"`
	Side              string `gorm:"column:side;type:varchar(10);not null;comment:交易方向"`
	Quantity          string `gorm:"column:quantity;type:decimal(20,8);not null;comment:交易数量"`
	Price             string `gorm:"column:price;type:decimal(20,8);not null;comment:交易价格"`
	RiskLevel         string `gorm:"column:risk_level;type:varchar(20);not null;comment:风险等级"`
	RiskScore         string `gorm:"column:risk_score;type:decimal(5,2);not null;comment:量化风险分"`
	MarginRequirement string `gorm:"column:margin_requirement;type:decimal(20,8);not null;comment:保证金要求"`
	IsAllowed         bool   `gorm:"column:is_allowed;type:boolean;not null;comment:是否允许交易"`
	Reason            string `gorm:"column:reason;type:text;comment:判定原因"`
}

func (RiskAssessmentModel) TableName() string {
	return "risk_assessments"
}

func (m *RiskAssessmentModel) ToDomain() *domain.RiskAssessment {
	q, err := decimal.NewFromString(m.Quantity)
	if err != nil {
		q = decimal.Zero
	}
	p, err := decimal.NewFromString(m.Price)
	if err != nil {
		p = decimal.Zero
	}
	rs, err := decimal.NewFromString(m.RiskScore)
	if err != nil {
		rs = decimal.Zero
	}
	mr, err := decimal.NewFromString(m.MarginRequirement)
	if err != nil {
		mr = decimal.Zero
	}
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

// RiskMetricsModel 用户量化风险指标数据库模型。
type RiskMetricsModel struct {
	gorm.Model
	UserID      string `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null;comment:用户ID"`
	VaR95       string `gorm:"column:var_95;type:decimal(20,8);not null;comment:95置信度VaR"`
	VaR99       string `gorm:"column:var_99;type:decimal(20,8);not null;comment:99置信度VaR"`
	MaxDrawdown string `gorm:"column:max_drawdown;type:decimal(20,8);not null;comment:历史最大回撤"`
	SharpeRatio string `gorm:"column:sharpe_ratio;type:decimal(20,8);not null;comment:夏普比率"`
	Correlation string `gorm:"column:correlation;type:decimal(20,8);not null;comment:组合相关性"`
}

func (RiskMetricsModel) TableName() string {
	return "risk_metrics"
}

func (m *RiskMetricsModel) ToDomain() *domain.RiskMetrics {
	v95, err := decimal.NewFromString(m.VaR95)
	if err != nil {
		v95 = decimal.Zero
	}
	v99, err := decimal.NewFromString(m.VaR99)
	if err != nil {
		v99 = decimal.Zero
	}
	md, err := decimal.NewFromString(m.MaxDrawdown)
	if err != nil {
		md = decimal.Zero
	}
	sr, err := decimal.NewFromString(m.SharpeRatio)
	if err != nil {
		sr = decimal.Zero
	}
	corr, err := decimal.NewFromString(m.Correlation)
	if err != nil {
		corr = decimal.Zero
	}
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

// RiskLimitModel 风险限额配置数据库模型。
type RiskLimitModel struct {
	gorm.Model
	ID           string `gorm:"column:id;type:varchar(36);primaryKey"`
	UserID       string `gorm:"column:user_id;type:varchar(36);index;not null;comment:用户ID"`
	LimitType    string `gorm:"column:limit_type;type:varchar(50);not null;comment:限额类型"`
	LimitValue   string `gorm:"column:limit_value;type:decimal(20,8);not null;comment:限定阈值"`
	CurrentValue string `gorm:"column:current_value;type:decimal(20,8);not null;comment:当前已用额度"`
	IsExceeded   bool   `gorm:"column:is_exceeded;type:boolean;not null;comment:是否已超限"`
}

func (RiskLimitModel) TableName() string {
	return "risk_limits"
}

func (m *RiskLimitModel) ToDomain() *domain.RiskLimit {
	lv, err := decimal.NewFromString(m.LimitValue)
	if err != nil {
		lv = decimal.Zero
	}
	cv, err := decimal.NewFromString(m.CurrentValue)
	if err != nil {
		cv = decimal.Zero
	}
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

// RiskAlertModel 风险告警记录数据库模型。
type RiskAlertModel struct {
	gorm.Model
	ID        string `gorm:"column:id;type:varchar(36);primaryKey"`
	UserID    string `gorm:"column:user_id;type:varchar(36);index;not null;comment:受影响用户"`
	AlertType string `gorm:"column:alert_type;type:varchar(50);not null;comment:告警类型"`
	Severity  string `gorm:"column:severity;type:varchar(20);not null;comment:严重程度"`
	Message   string `gorm:"column:message;type:text;not null;comment:告警描述"`
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

// CircuitBreakerModel 风险熔断状态数据库模型。
type CircuitBreakerModel struct {
	gorm.Model
	UserID        string     `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null;comment:用户ID"`
	IsFired       bool       `gorm:"column:is_fired;type:boolean;not null;comment:熔断器是否触发"`
	TriggerReason string     `gorm:"column:trigger_reason;type:text;comment:触发原因"`
	FiredAt       *time.Time `gorm:"column:fired_at;type:datetime;comment:触发时间"`
	AutoResetAt   *time.Time `gorm:"column:auto_reset_at;type:datetime;comment:预期的自动恢复时间"`
}

func (CircuitBreakerModel) TableName() string {
	return "risk_circuit_breakers"
}

func (m *CircuitBreakerModel) ToDomain() *domain.CircuitBreaker {
	return &domain.CircuitBreaker{
		Model:         m.Model,
		UserID:        m.UserID,
		IsFired:       m.IsFired,
		TriggerReason: m.TriggerReason,
		FiredAt:       m.FiredAt,
		AutoResetAt:   m.AutoResetAt,
	}
}

func FromCircuitBreakerDomain(d *domain.CircuitBreaker) *CircuitBreakerModel {
	return &CircuitBreakerModel{
		Model:         d.Model,
		UserID:        d.UserID,
		IsFired:       d.IsFired,
		TriggerReason: d.TriggerReason,
		FiredAt:       d.FiredAt,
		AutoResetAt:   d.AutoResetAt,
	}
}

// assessmentRepository 实现了评估记录仓储接口。
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

// metricsRepository 实现了风险指标仓储接口。
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

// limitRepository 实现了限额配置仓储接口。
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

// alertRepository 实现了告警历史仓储接口。
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

// circuitBreakerRepository 实现了熔断状态仓储接口。
type circuitBreakerRepository struct {
	db *gorm.DB
}

func NewCircuitBreakerRepository(db *gorm.DB) domain.CircuitBreakerRepository {
	return &circuitBreakerRepository{db: db}
}

func (r *circuitBreakerRepository) Save(ctx context.Context, cb *domain.CircuitBreaker) error {
	model := FromCircuitBreakerDomain(cb)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	cb.Model = model.Model
	return nil
}

func (r *circuitBreakerRepository) GetByUserID(ctx context.Context, userID string) (*domain.CircuitBreaker, error) {
	var model CircuitBreakerModel
	if err := r.db.WithContext(ctx).First(&model, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}
