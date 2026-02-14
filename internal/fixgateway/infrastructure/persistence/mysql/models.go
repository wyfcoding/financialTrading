package mysql

import (
	"time"

	"github.com/wyfcoding/financialtrading/internal/fixgateway/domain"
	"gorm.io/gorm"
)

// FixSessionModel FIX 会话模型 (数据库映射)
type FixSessionModel struct {
	gorm.Model
	SessionID     string    `gorm:"column:session_id;type:varchar(64);uniqueIndex;not null"`
	CompID        string    `gorm:"column:comp_id;type:varchar(32);not null"`
	TargetID      string    `gorm:"column:target_id;type:varchar(32);not null"`
	FixVersion    string    `gorm:"column:fix_version;type:varchar(16)"`
	Status        string    `gorm:"column:status;type:varchar(16)"`
	LastMsgSeqIn  int       `gorm:"column:last_msg_seq_in"`
	LastMsgSeqOut int       `gorm:"column:last_msg_seq_out"`
	LastActiveAt  time.Time `gorm:"column:last_active_at"`
}

func (FixSessionModel) TableName() string {
	return "fix_sessions"
}

func toDomainSession(m *FixSessionModel) *domain.FixSession {
	return &domain.FixSession{
		SessionID:     m.SessionID,
		CompID:        m.CompID,
		TargetID:      m.TargetID,
		Version:       domain.FixVersion(m.FixVersion),
		Status:        domain.FixSessionStatus(m.Status),
		LastMsgSeqIn:  m.LastMsgSeqIn,
		LastMsgSeqOut: m.LastMsgSeqOut,
		LastActiveAt:  m.LastActiveAt,
	}
}
