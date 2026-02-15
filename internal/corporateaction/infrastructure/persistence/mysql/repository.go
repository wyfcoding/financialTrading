// Package mysql 公司行动 MySQL 仓储实现
package mysql

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/corporateaction/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/database"
	"gorm.io/gorm"
)

type ActionRepositoryImpl struct {
	db *database.DB
}

func NewActionRepository(db *database.DB) domain.ActionRepository {
	return &ActionRepositoryImpl{db: db}
}

func (r *ActionRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return r.db.DB.WithContext(ctx)
}

func (r *ActionRepositoryImpl) Save(ctx context.Context, action *domain.CorporateAction) error {
	return r.getDB(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(action).Error
}

func (r *ActionRepositoryImpl) GetByEventID(ctx context.Context, eventID string) (*domain.CorporateAction, error) {
	var action domain.CorporateAction
	err := r.getDB(ctx).Preload("Entitlements").Where("event_id = ?", eventID).First(&action).Error
	return &action, err
}

func (r *ActionRepositoryImpl) ListActive(ctx context.Context, date time.Time) ([]*domain.CorporateAction, error) {
	var actions []*domain.CorporateAction
	// 查找 ExDate <= date 且未完成的事件
	err := r.getDB(ctx).Where("status IN ? AND ex_date <= ?", []string{"ANNOUNCED", "ACTIVE"}, date).Find(&actions).Error
	return actions, err
}

type EntitlementRepositoryImpl struct {
	db *database.DB
}

func NewEntitlementRepository(db *database.DB) domain.EntitlementRepository {
	return &EntitlementRepositoryImpl{db: db}
}

func (r *EntitlementRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return r.db.DB.WithContext(ctx)
}

func (r *EntitlementRepositoryImpl) Save(ctx context.Context, ent *domain.Entitlement) error {
	return r.getDB(ctx).Save(ent).Error
}

func (r *EntitlementRepositoryImpl) ListByActionID(ctx context.Context, actionID uint) ([]*domain.Entitlement, error) {
	var ents []*domain.Entitlement
	err := r.getDB(ctx).Where("action_id = ?", actionID).Find(&ents).Error
	return ents, err
}

func (r *EntitlementRepositoryImpl) GetByAccountAndAction(ctx context.Context, accountID string, actionID uint) (*domain.Entitlement, error) {
	var ent domain.Entitlement
	err := r.getDB(ctx).Where("account_id = ? AND action_id = ?", accountID, actionID).First(&ent).Error
	return &ent, err
}
