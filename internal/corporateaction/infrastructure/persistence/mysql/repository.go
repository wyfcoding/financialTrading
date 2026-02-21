package mysql

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/corporateaction/domain"
	"gorm.io/gorm"
)

type actionRepository struct {
	db *gorm.DB
}

// NewActionRepository 初始化基于 MySQL 的行动记录仓储
func NewActionRepository(db *gorm.DB) domain.ActionRepository {
	_ = db.AutoMigrate(&domain.CorporateAction{})
	return &actionRepository{db: db}
}

func (r *actionRepository) Save(ctx context.Context, action *domain.CorporateAction) error {
	return r.db.WithContext(ctx).Save(action).Error
}

func (r *actionRepository) GetByEventID(ctx context.Context, eventID string) (*domain.CorporateAction, error) {
	var action domain.CorporateAction
	if err := r.db.WithContext(ctx).Where("event_id = ?", eventID).First(&action).Error; err != nil {
		return nil, err
	}
	return &action, nil
}

func (r *actionRepository) ListActive(ctx context.Context, date time.Time) ([]*domain.CorporateAction, error) {
	var actions []*domain.CorporateAction
	// 查找在除权日之前或者在派发日附近的所有需要处理的活动
	if err := r.db.WithContext(ctx).Where("status != ?", domain.ActionStatusCompleted).Find(&actions).Error; err != nil {
		return nil, err
	}
	return actions, nil
}

type entitlementRepository struct {
	db *gorm.DB
}

// NewEntitlementRepository 初始化权益明细的 MySQL 仓储
func NewEntitlementRepository(db *gorm.DB) domain.EntitlementRepository {
	_ = db.AutoMigrate(&domain.Entitlement{})
	return &entitlementRepository{db: db}
}

func (r *entitlementRepository) Save(ctx context.Context, ent *domain.Entitlement) error {
	return r.db.WithContext(ctx).Save(ent).Error
}

func (r *entitlementRepository) ListByActionID(ctx context.Context, actionID uint) ([]*domain.Entitlement, error) {
	var ents []*domain.Entitlement
	if err := r.db.WithContext(ctx).Where("action_id = ?", actionID).Find(&ents).Error; err != nil {
		return nil, err
	}
	return ents, nil
}

func (r *entitlementRepository) GetByAccountAndAction(ctx context.Context, accountID string, actionID uint) (*domain.Entitlement, error) {
	var ent domain.Entitlement
	if err := r.db.WithContext(ctx).Where("account_id = ? AND action_id = ?", accountID, actionID).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}
