package domain

import (
	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	StatusPending    ApplicationStatus = "PENDING"
	StatusProcessing ApplicationStatus = "PROCESSING"
	StatusApproved   ApplicationStatus = "APPROVED"
	StatusRejected   ApplicationStatus = "REJECTED"
)

// OnboardingApplication 开户申请实体
type OnboardingApplication struct {
	gorm.Model
	ApplicationID string            `gorm:"column:application_id;type:varchar(32);unique_index;not null"`
	FirstName     string            `gorm:"column:first_name;type:varchar(50);not null"`
	LastName      string            `gorm:"column:last_name;type:varchar(50);not null"`
	Email         string            `gorm:"column:email;type:varchar(100);index;not null"`
	IDNumber      string            `gorm:"column:id_number;type:varchar(50);not null"`
	Address       string            `gorm:"column:address;type:varchar(255)"`
	Status        ApplicationStatus `gorm:"column:status;type:varchar(20);not null;default:'PENDING'"`
	KYCStatus     string            `gorm:"column:kyc_status;type:varchar(20);default:'NONE'"`
}

func (OnboardingApplication) TableName() string { return "onboarding_applications" }
