package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/fixgateway/domain"
	"gorm.io/gorm"
)

type GormFixRepository struct {
	db *gorm.DB
}

func NewGormFixRepository(db *gorm.DB) domain.FixRepository {
	return &GormFixRepository{db: db}
}

func (r *GormFixRepository) GetSession(ctx context.Context, sessionID string) (*domain.FixSession, error) {
	var model FixSessionModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&model).Error; err != nil {
		return nil, err
	}
	return toDomainSession(&model), nil
}

func (r *GormFixRepository) SaveSession(ctx context.Context, session *domain.FixSession) error {
	model := FixSessionModel{
		SessionID:     session.SessionID,
		CompID:        session.CompID,
		TargetID:      session.TargetID,
		FixVersion:    session.Version,
		Status:        string(session.Status),
		LastMsgSeqIn:  session.LastMsgSeqIn,
		LastMsgSeqOut: session.LastMsgSeqOut,
		LastActiveAt:  session.LastActiveAt,
	}

	// 首先尝试查找是否存在记录以获取 ID (用于更新)
	var existing FixSessionModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", session.SessionID).First(&existing).Error; err == nil {
		model.ID = existing.ID
		model.CreatedAt = existing.CreatedAt
	}

	return r.db.WithContext(ctx).Save(&model).Error
}
