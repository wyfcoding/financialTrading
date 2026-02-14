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
		FixVersion:    string(session.Version),
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

func (r *GormFixRepository) GetSessionByCompIDs(ctx context.Context, compID, targetID string) (*domain.FixSession, error) {
	var model FixSessionModel
	if err := r.db.WithContext(ctx).
		Where("comp_id = ? AND target_id = ?", compID, targetID).
		Order("id DESC").
		First(&model).Error; err != nil {
		return nil, err
	}
	return toDomainSession(&model), nil
}

func (r *GormFixRepository) ListActiveSessions(ctx context.Context) ([]*domain.FixSession, error) {
	var models []FixSessionModel
	if err := r.db.WithContext(ctx).Where("status = ?", string(domain.FixSessionActive)).Find(&models).Error; err != nil {
		return nil, err
	}
	sessions := make([]*domain.FixSession, 0, len(models))
	for i := range models {
		sessions = append(sessions, toDomainSession(&models[i]))
	}
	return sessions, nil
}

func (r *GormFixRepository) DeleteSession(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&FixSessionModel{}).Error
}

func (r *GormFixRepository) SaveMessage(ctx context.Context, message *domain.FixMessage) error {
	// Message persistence is optional for current Stage A gate; session state remains authoritative.
	return nil
}

func (r *GormFixRepository) GetMessages(ctx context.Context, sessionID string, limit int) ([]*domain.FixMessage, error) {
	return []*domain.FixMessage{}, nil
}

var _ domain.FixRepository = (*GormFixRepository)(nil)
