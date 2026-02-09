package domain

import (
	"gorm.io/gorm"
)

// AMLRecord AML检查历史记录
type AMLRecord struct {
	gorm.Model
	UserID    uint64 `gorm:"column:user_id;index;not null"`
	Name      string `gorm:"column:name;type:varchar(100)"`
	Country   string `gorm:"column:country;type:varchar(100)"`
	Passed    bool   `gorm:"column:passed;not null"`
	RiskLevel string `gorm:"column:risk_level;type:varchar(20)"`
	Reason    string `gorm:"column:reason;type:text"`
}

func (AMLRecord) TableName() string { return "aml_records" }

// AMLAlert 反洗钱警报实体
type AMLAlert struct {
	gorm.Model
	AlertID     string `gorm:"column:alert_id;type:varchar(32);uniqueIndex;not null"`
	UserID      uint64 `gorm:"column:user_id;index;not null"`
	Type        string `gorm:"column:type;type:varchar(50);not null"`
	Description string `gorm:"column:description;type:text"`
	Status      string `gorm:"column:status;type:varchar(20);not null;default:'PENDING'"`
}

func (AMLAlert) TableName() string { return "aml_alerts" }

// UserRiskScore 用户风险评分实体
type UserRiskScore struct {
	gorm.Model
	UserID    uint64  `gorm:"column:user_id;uniqueIndex;not null"`
	Score     float64 `gorm:"column:score;not null"`
	RiskLevel string  `gorm:"column:risk_level;type:varchar(20);not null"`
}

func (UserRiskScore) TableName() string { return "user_risk_scores" }
