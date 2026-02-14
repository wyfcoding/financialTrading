//go:build ignore

package domain

import (
	"gorm.io/gorm"
)

// UserRiskScore 用户风险评分实体
type UserRiskScore struct {
	gorm.Model
	UserID    string  `gorm:"column:user_id;type:varchar(32);unique_index;not null"`
	Score     float64 `gorm:"column:score;not null"`
	RiskLevel string  `gorm:"column:risk_level;type:varchar(20);not null"`
}

func (UserRiskScore) TableName() string { return "user_risk_scores" }
