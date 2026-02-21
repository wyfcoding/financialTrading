// Package mysql 市场监察服务 MySQL 持久化实现
// 生成摘要：
//  1. 实现 AlertRepository / RuleRepository / UserScoreRepository / OrderEventRepository
//  2. 使用 GORM 操作 MySQL
//  3. 支持分页、复合过滤查询
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
)

// AlertRepository 告警仓储实现
type AlertRepository struct {
	db *gorm.DB
}

// NewAlertRepository 创建告警仓储
func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// Save 保存或更新告警
func (r *AlertRepository) Save(ctx context.Context, alert *domain.SurveillanceAlert) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "alert_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "severity", "reviewer_id", "review_comment", "reviewed_at", "updated_at"}),
		}).
		Create(alert).Error
}

// GetByID 根据 alertID 查询告警
func (r *AlertRepository) GetByID(ctx context.Context, alertID string) (*domain.SurveillanceAlert, error) {
	var alert domain.SurveillanceAlert
	if err := r.db.WithContext(ctx).Where("alert_id = ?", alertID).First(&alert).Error; err != nil {
		return nil, err
	}
	return &alert, nil
}

// ListByStatus 按状态列表查询
func (r *AlertRepository) ListByStatus(
	ctx context.Context,
	status domain.AlertStatus,
	page, pageSize int,
) ([]*domain.SurveillanceAlert, int64, error) {
	var alerts []*domain.SurveillanceAlert
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.SurveillanceAlert{}).Where("status = ?", status)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("detected_at DESC").Find(&alerts).Error; err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

// ListByUserID 按用户查询
func (r *AlertRepository) ListByUserID(
	ctx context.Context,
	userID string,
	page, pageSize int,
) ([]*domain.SurveillanceAlert, int64, error) {
	var alerts []*domain.SurveillanceAlert
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.SurveillanceAlert{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("detected_at DESC").Find(&alerts).Error; err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

// ListByFilters 综合条件查询
func (r *AlertRepository) ListByFilters(
	ctx context.Context,
	filters domain.AlertFilters,
	page, pageSize int,
) ([]*domain.SurveillanceAlert, int64, error) {
	var alerts []*domain.SurveillanceAlert
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.SurveillanceAlert{})
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.Severity != nil {
		query = query.Where("severity = ?", *filters.Severity)
	}
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Symbol != "" {
		query = query.Where("symbol = ?", filters.Symbol)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("detected_at DESC").Find(&alerts).Error; err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

// RuleRepository 规则仓储实现
type RuleRepository struct {
	db *gorm.DB
}

// NewRuleRepository 创建规则仓储
func NewRuleRepository(db *gorm.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

// Save 保存或更新规则
func (r *RuleRepository) Save(ctx context.Context, rule *domain.SurveillanceRule) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "rule_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "enabled", "window_seconds", "threshold", "min_cancel_ratio", "max_wash_volume_ratio", "price_deviation_threshold", "description", "updated_at"}),
		}).
		Create(rule).Error
}

// GetByID 查询规则
func (r *RuleRepository) GetByID(ctx context.Context, ruleID string) (*domain.SurveillanceRule, error) {
	var rule domain.SurveillanceRule
	if err := r.db.WithContext(ctx).Where("rule_id = ?", ruleID).First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListByType 按类型查询
func (r *RuleRepository) ListByType(ctx context.Context, manipType domain.ManipulationType) ([]*domain.SurveillanceRule, error) {
	var rules []*domain.SurveillanceRule
	if err := r.db.WithContext(ctx).Where("type = ?", manipType).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListEnabled 查询启用的规则
func (r *RuleRepository) ListEnabled(ctx context.Context) ([]*domain.SurveillanceRule, error) {
	var rules []*domain.SurveillanceRule
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListAll 查询全部规则
func (r *RuleRepository) ListAll(ctx context.Context) ([]*domain.SurveillanceRule, error) {
	var rules []*domain.SurveillanceRule
	if err := r.db.WithContext(ctx).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// UserScoreRepository 用户评分仓储实现
type UserScoreRepository struct {
	db *gorm.DB
}

// NewUserScoreRepository 创建评分仓储
func NewUserScoreRepository(db *gorm.DB) *UserScoreRepository {
	return &UserScoreRepository{db: db}
}

// Save 保存或更新评分
func (r *UserScoreRepository) Save(ctx context.Context, score *domain.UserSurveillanceScore) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"overall_score", "spoofing_score", "wash_trading_score", "layering_score", "total_alerts", "confirmed_alerts", "updated_at"}),
		}).
		Create(score).Error
}

// GetByUserID 获取用户评分
func (r *UserScoreRepository) GetByUserID(ctx context.Context, userID string) (*domain.UserSurveillanceScore, error) {
	var score domain.UserSurveillanceScore
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&score).Error; err != nil {
		return nil, err
	}
	return &score, nil
}

// OrderEventRepository 订单事件仓储实现
type OrderEventRepository struct {
	db *gorm.DB
}

// NewOrderEventRepository 创建事件仓储
func NewOrderEventRepository(db *gorm.DB) *OrderEventRepository {
	return &OrderEventRepository{db: db}
}

// SaveEvent 保存单个事件
func (r *OrderEventRepository) SaveEvent(ctx context.Context, event *domain.OrderEventRecord) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// SaveBatch 批量保存
func (r *OrderEventRepository) SaveBatch(ctx context.Context, events []*domain.OrderEventRecord) error {
	return r.db.WithContext(ctx).CreateInBatches(events, 100).Error
}

// GetByUserAndSymbol 按用户和标的查询时间范围内事件
func (r *OrderEventRepository) GetByUserAndSymbol(
	ctx context.Context,
	userID, symbol string,
	start, end interface{},
) ([]*domain.OrderEventRecord, error) {
	var records []*domain.OrderEventRecord
	query := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol = ?", userID, symbol)

	if s, ok := start.(time.Time); ok {
		query = query.Where("event_time >= ?", s)
	}
	if e, ok := end.(time.Time); ok {
		query = query.Where("event_time <= ?", e)
	}

	if err := query.Order("event_time ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// GetByUser 按用户查询时间范围内事件
func (r *OrderEventRepository) GetByUser(
	ctx context.Context,
	userID string,
	start, end interface{},
) ([]*domain.OrderEventRecord, error) {
	var records []*domain.OrderEventRecord
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if s, ok := start.(time.Time); ok {
		query = query.Where("event_time >= ?", s)
	}
	if e, ok := end.(time.Time); ok {
		query = query.Where("event_time <= ?", e)
	}

	if err := query.Order("event_time ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
