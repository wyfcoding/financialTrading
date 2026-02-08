// Package domain 合规服务领域层
// 生成摘要：
// 1) 定义 KYC 申请聚合根
// 2) 定义 AML/风险评估领域模型
package domain

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// KYCStatus KYC状态
type KYCStatus int8

const (
	KYCStatusPending          KYCStatus = 1 // 待审核
	KYCStatusApproved         KYCStatus = 2 // 已通过
	KYCStatusRejected         KYCStatus = 3 // 已拒绝
	KYCStatusMoreInfoRequired KYCStatus = 4 // 需补充材料
)

// KYCLevel KYC等级
type KYCLevel int8

const (
	KYCLevel1 KYCLevel = 1 // 基础认证
	KYCLevel2 KYCLevel = 2 // 高级认证
	KYCLevel3 KYCLevel = 3 // 机构认证
)

// KYCApplication KYC申请聚合根
type KYCApplication struct {
	gorm.Model
	ApplicationID string    `gorm:"column:application_id;type:varchar(32);unique_index;not null"`
	UserID        uint64    `gorm:"column:user_id;unique_index;not null"`
	Level         KYCLevel  `gorm:"column:level;type:tinyint;not null"`
	Status        KYCStatus `gorm:"column:status;type:tinyint;not null;default:1"`

	// 个人信息
	FirstName   string `gorm:"column:first_name;type:varchar(64)"`
	LastName    string `gorm:"column:last_name;type:varchar(64)"`
	IDNumber    string `gorm:"column:id_number;type:varchar(64)"`
	DateOfBirth string `gorm:"column:date_of_birth;type:varchar(20)"`
	Country     string `gorm:"column:country;type:varchar(32)"`

	// 文件链接
	IDCardFrontURL string `gorm:"column:id_card_front_url;type:varchar(512)"`
	IDCardBackURL  string `gorm:"column:id_card_back_url;type:varchar(512)"`
	FacePhotoURL   string `gorm:"column:face_photo_url;type:varchar(512)"`

	// 审核信息
	RejectReason string     `gorm:"column:reject_reason;type:varchar(255)"`
	ReviewerID   string     `gorm:"column:reviewer_id;type:varchar(64)"`
	ReviewedAt   *time.Time `gorm:"column:reviewed_at"`

	// 领域事件
	domainEvents []DomainEvent `gorm:"-"`
}

// TableName 表名
func (KYCApplication) TableName() string {
	return "kyc_applications"
}

// NewKYCApplication 创建KYC申请
func NewKYCApplication(userID uint64, level KYCLevel, first, last, idNum, dob, country string, front, back, face string) *KYCApplication {
	return &KYCApplication{
		UserID:         userID,
		Level:          level,
		Status:         KYCStatusPending,
		FirstName:      first,
		LastName:       last,
		IDNumber:       idNum,
		DateOfBirth:    dob,
		Country:        country,
		IDCardFrontURL: front,
		IDCardBackURL:  back,
		FacePhotoURL:   face,
		domainEvents:   make([]DomainEvent, 0),
	}
}

// Approve 通过审核
func (k *KYCApplication) Approve(reviewerID string) error {
	if k.Status != KYCStatusPending && k.Status != KYCStatusMoreInfoRequired {
		return errors.New("invalid status for approval")
	}

	now := time.Now()
	k.Status = KYCStatusApproved
	k.ReviewerID = reviewerID
	k.ReviewedAt = &now

	k.addEvent(&KYCApprovedEvent{
		ApplicationID: k.ApplicationID,
		UserID:        k.UserID,
		Level:         k.Level,
		ReviewerID:    reviewerID,
		Timestamp:     now,
	})

	return nil
}

// Reject 拒绝审核
func (k *KYCApplication) Reject(reviewerID, reason string) error {
	if k.Status != KYCStatusPending && k.Status != KYCStatusMoreInfoRequired {
		return errors.New("invalid status for rejection")
	}

	now := time.Now()
	k.Status = KYCStatusRejected
	k.ReviewerID = reviewerID
	k.RejectReason = reason
	k.ReviewedAt = &now

	k.addEvent(&KYCRejectedEvent{
		ApplicationID: k.ApplicationID,
		UserID:        k.UserID,
		Reason:        reason,
		ReviewerID:    reviewerID,
		Timestamp:     now,
	})

	return nil
}

// RequireMoreInfo 要求补充材料
func (k *KYCApplication) RequireMoreInfo(reviewerID, note string) error {
	// ... 类似逻辑
	k.Status = KYCStatusMoreInfoRequired
	return nil
}

func (k *KYCApplication) addEvent(event DomainEvent) {
	k.domainEvents = append(k.domainEvents, event)
}

func (k *KYCApplication) GetDomainEvents() []DomainEvent {
	return k.domainEvents
}

func (k *KYCApplication) ClearDomainEvents() {
	k.domainEvents = nil
}

// AMLRecord AML检查记录
type AMLRecord struct {
	gorm.Model
	UserID    uint64 `gorm:"column:user_id;index;not null"`
	Name      string `gorm:"column:name;type:varchar(128)"`
	Country   string `gorm:"column:country;type:varchar(32)"`
	Passed    bool   `gorm:"column:passed;not null"`
	RiskLevel string `gorm:"column:risk_level;type:varchar(16)"`
	Reason    string `gorm:"column:reason;type:varchar(255)"`
}

// TableName 表名
func (AMLRecord) TableName() string {
	return "aml_records"
}
