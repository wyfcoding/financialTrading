package domain

import "gorm.io/gorm"

type UserProfile struct {
	gorm.Model
	Email string `gorm:"column:email;type:varchar(255);uniqueIndex;not null"`
	Name  string `gorm:"column:name;type:varchar(100)"`
	Phone string `gorm:"column:phone;type:varchar(20)"`
}

func (UserProfile) TableName() string { return "user_profiles" }
