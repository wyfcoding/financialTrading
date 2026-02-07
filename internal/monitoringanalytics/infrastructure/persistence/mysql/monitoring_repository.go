// Package mysql 提供了监控分析服务指标与系统健康仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"github.com/wyfcoding/pkg/contextx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type metricRepository struct {
	db *gorm.DB
}

func NewMetricRepository(db *gorm.DB) domain.MetricRepository {
	return &metricRepository{db: db}
}

// --- tx helpers ---

func (r *metricRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *metricRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *metricRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *metricRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *metricRepository) Save(ctx context.Context, m *domain.Metric) error {
	model, err := toMetricModel(m)
	if err != nil {
		return err
	}
	if model == nil {
		return nil
	}

	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error; err != nil {
			return err
		}
		m.ID = model.ID
		m.CreatedAt = model.CreatedAt
		m.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&MetricModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"name":      model.Name,
			"value":     model.Value,
			"tags":      model.TagsJSON,
			"timestamp": model.Timestamp,
		}).Error
}

func (r *metricRepository) GetMetrics(ctx context.Context, name string, startTime, endTime int64) ([]*domain.Metric, error) {
	var models []MetricModel
	if err := r.getDB(ctx).WithContext(ctx).
		Where("name = ? AND timestamp >= ? AND timestamp <= ?", name, startTime, endTime).
		Order("timestamp asc").
		Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Metric, len(models))
	for i := range models {
		metric, err := toMetric(&models[i])
		if err != nil {
			return nil, err
		}
		res[i] = metric
	}
	return res, nil
}

func (r *metricRepository) GetTradeMetrics(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*domain.TradeMetric, error) {
	var models []TradeMetricModel
	if err := r.getDB(ctx).WithContext(ctx).
		Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, startTime, endTime).
		Order("timestamp asc").
		Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.TradeMetric, len(models))
	for i := range models {
		res[i] = toTradeMetric(&models[i])
	}
	return res, nil
}

func (r *metricRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// --- SystemHealth ---

type systemHealthRepository struct {
	db *gorm.DB
}

func NewSystemHealthRepository(db *gorm.DB) domain.SystemHealthRepository {
	return &systemHealthRepository{db: db}
}

func (r *systemHealthRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *systemHealthRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *systemHealthRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *systemHealthRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *systemHealthRepository) Save(ctx context.Context, h *domain.SystemHealth) error {
	model := toHealthModel(h)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		h.ID = model.ID
		h.CreatedAt = model.CreatedAt
		h.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&SystemHealthModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"service_name": model.ServiceName,
			"status":       model.Status,
			"cpu_usage":    model.CPUUsage,
			"memory_usage": model.MemoryUsage,
			"message":      model.Message,
			"last_checked": model.LastChecked,
		}).Error
}

func (r *systemHealthRepository) GetLatestHealth(ctx context.Context, serviceName string, limit int) ([]*domain.SystemHealth, error) {
	var models []SystemHealthModel
	query := r.getDB(ctx).WithContext(ctx)
	if serviceName != "" {
		query = query.Where("service_name = ?", serviceName)
	}
	if err := query.Order("last_checked desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.SystemHealth, len(models))
	for i := range models {
		res[i] = toHealth(&models[i])
	}
	return res, nil
}

func (r *systemHealthRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// --- Alert ---

type alertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) domain.AlertRepository {
	return &alertRepository{db: db}
}

func (r *alertRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *alertRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *alertRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *alertRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *alertRepository) Save(ctx context.Context, a *domain.Alert) error {
	model := toAlertModel(a)
	if model == nil {
		return nil
	}
	db := r.getDB(ctx).WithContext(ctx)
	if model.ID == 0 {
		if err := db.Create(model).Error; err != nil {
			return err
		}
		a.ID = model.ID
		a.CreatedAt = model.CreatedAt
		a.UpdatedAt = model.UpdatedAt
		return nil
	}
	return db.Model(&AlertModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"alert_id":  model.AlertID,
			"rule_name": model.RuleName,
			"severity":  model.Severity,
			"message":   model.Message,
			"source":    model.Source,
			"status":    model.Status,
			"timestamp": model.Timestamp,
		}).Error
}

func (r *alertRepository) UpdateStatus(ctx context.Context, alertID, status string) error {
	if alertID == "" {
		return nil
	}
	return r.getDB(ctx).WithContext(ctx).
		Model(&AlertModel{}).
		Where("alert_id = ?", alertID).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (r *alertRepository) GetAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	var models []AlertModel
	if err := r.getDB(ctx).WithContext(ctx).Order("timestamp desc").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.Alert, len(models))
	for i := range models {
		res[i] = toAlert(&models[i])
	}
	return res, nil
}

func (r *alertRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}
