package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/compliance/domain"
	"gorm.io/gorm"
)

type KYCRepo struct {
	db *gorm.DB
}

func NewKYCRepo(db *gorm.DB) domain.KYCRepository {
	return &KYCRepo{db: db}
}

func (r *KYCRepo) Save(ctx context.Context, kyc *domain.KYCApplication) error {
	// First check if exists to update ID
	if kyc.ID == 0 {
		var existing domain.KYCApplication
		if err := r.db.WithContext(ctx).Where("application_id = ?", kyc.ApplicationID).First(&existing).Error; err == nil {
			kyc.ID = existing.ID
			kyc.CreatedAt = existing.CreatedAt
		}
	}
	return r.db.WithContext(ctx).Save(kyc).Error
}

func (r *KYCRepo) GetByUserID(ctx context.Context, userID uint64) (*domain.KYCApplication, error) {
	var kyc domain.KYCApplication
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&kyc).Error; err != nil {
		return nil, err
	}
	return &kyc, nil
}

func (r *KYCRepo) GetByApplicationID(ctx context.Context, appID string) (*domain.KYCApplication, error) {
	var kyc domain.KYCApplication
	if err := r.db.WithContext(ctx).Where("application_id = ?", appID).First(&kyc).Error; err != nil {
		return nil, err
	}
	return &kyc, nil
}

func (r *KYCRepo) GetPending(ctx context.Context, limit int) ([]*domain.KYCApplication, error) {
	var kycs []*domain.KYCApplication
	err := r.db.WithContext(ctx).
		Where("status = ?", domain.KYCStatusPending).
		Order("created_at asc").
		Limit(limit).
		Find(&kycs).Error
	return kycs, err
}

type AMLRepo struct {
	db *gorm.DB
}

func NewAMLRepo(db *gorm.DB) domain.AMLRepository {
	return &AMLRepo{db: db}
}

func (r *AMLRepo) Save(ctx context.Context, record *domain.AMLRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *AMLRepo) GetLatestByUserID(ctx context.Context, userID uint64) (*domain.AMLRecord, error) {
	var record domain.AMLRecord
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AMLRepo) SaveAlert(ctx context.Context, alert *domain.AMLAlert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

func (r *AMLRepo) ListAlertsByStatus(ctx context.Context, status string) ([]*domain.AMLAlert, error) {
	var alerts []*domain.AMLAlert
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&alerts).Error
	return alerts, err
}

func (r *AMLRepo) SaveRiskScore(ctx context.Context, score *domain.UserRiskScore) error {
	var existing domain.UserRiskScore
	if err := r.db.WithContext(ctx).Where("user_id = ?", score.UserID).First(&existing).Error; err == nil {
		score.ID = existing.ID
		score.CreatedAt = existing.CreatedAt
	}
	return r.db.WithContext(ctx).Save(score).Error
}

func (r *AMLRepo) GetRiskScore(ctx context.Context, userID uint64) (*domain.UserRiskScore, error) {
	var score domain.UserRiskScore
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&score).Error; err != nil {
		return nil, err
	}
	return &score, nil
}
