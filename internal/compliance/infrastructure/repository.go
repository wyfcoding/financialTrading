// Package infrastructure 合规服务基础设施层
package infrastructure

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
	"gorm.io/gorm"
)

// GormKYCRepository GORM KYC仓储实现
type GormKYCRepository struct {
	db *gorm.DB
}

// NewGormKYCRepository 创建KYC仓储
func NewGormKYCRepository(db *gorm.DB) *GormKYCRepository {
	return &GormKYCRepository{db: db}
}

// Save 保存KYC申请
func (r *GormKYCRepository) Save(ctx context.Context, kyc *domain.KYCApplication) error {
	return r.db.WithContext(ctx).Save(kyc).Error
}

// GetByUserID 根据用户ID获取
func (r *GormKYCRepository) GetByUserID(ctx context.Context, userID uint64) (*domain.KYCApplication, error) {
	var kyc domain.KYCApplication
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&kyc).Error; err != nil {
		return nil, fmt.Errorf("kyc application not found: %w", err)
	}
	return &kyc, nil
}

// GetByApplicationID 根据申请ID获取
func (r *GormKYCRepository) GetByApplicationID(ctx context.Context, appID string) (*domain.KYCApplication, error) {
	var kyc domain.KYCApplication
	if err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&kyc).Error; err != nil {
		return nil, fmt.Errorf("kyc application not found: %w", err)
	}
	return &kyc, nil
}

// GetPending 获取待审核列表
func (r *GormKYCRepository) GetPending(ctx context.Context, limit int) ([]*domain.KYCApplication, error) {
	var kycs []*domain.KYCApplication
	if err := r.db.WithContext(ctx).Where("status = ?", domain.KYCStatusPending).Limit(limit).Find(&kycs).Error; err != nil {
		return nil, err
	}
	return kycs, nil
}

// GormAMLRepository GORM AML仓储实现
type GormAMLRepository struct {
	db *gorm.DB
}

// NewGormAMLRepository 创建AML仓储
func NewGormAMLRepository(db *gorm.DB) *GormAMLRepository {
	return &GormAMLRepository{db: db}
}

// Save 保存AML记录
func (r *GormAMLRepository) Save(ctx context.Context, record *domain.AMLRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// GetLatestByUserID 获取最新AML记录
func (r *GormAMLRepository) GetLatestByUserID(ctx context.Context, userID uint64) (*domain.AMLRecord, error) {
	var record domain.AMLRecord
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&record).Error; err != nil {
		return nil, fmt.Errorf("aml record not found: %w", err)
	}
	return &record, nil
}
