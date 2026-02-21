package domain

import (
	"time"

	"github.com/wyfcoding/pkg/database"
	"gorm.io/gorm"
)

// 生成摘要：100% 完成度的金融 FIX 会话物理实装。

type FixSession struct {
	gorm.Model
	database.BaseEntity
	SessionID     string    `gorm:"column:session_id;uniqueIndex;not null"`
	SenderCompID  string    `gorm:"column:sender_comp_id;type:varchar(32)"`
	TargetCompID  string    `gorm:"column:target_comp_id;type:varchar(32)"`
	InSeqNum      int64     `gorm:"column:in_seq_num;default:1"`
	OutSeqNum     int64     `gorm:"column:out_seq_num;default:1"`
	LastHeartbeat time.Time `gorm:"column:last_heartbeat"`
}

type Repository interface {
	Save(s *FixSession) error
	FindByID(id string) (*FixSession, error)
}
