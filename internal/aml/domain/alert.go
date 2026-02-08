package domain

import (
	"gorm.io/gorm"
)

// AMLAlert 反洗钱警报实体
type AMLAlert struct {
	gorm.Model
	AlertID     string `gorm:"column:alert_id;type:varchar(32);unique_index;not null"`
	UserID      string `gorm:"column:user_id;type:varchar(32);index;not null"`
	Type        string `gorm:"column:type;type:varchar(50);not null"`
	Description string `gorm:"column:description;type:text"`
	Status      string `gorm:"column:status;type:varchar(20);not null;default:'PENDING'"`
}

func (AMLAlert) TableName() string { return "aml_alerts" }
