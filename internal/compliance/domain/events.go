// Package domain 合规服务领域事件
package domain

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// KYCApprovedEvent KYC通过事件
type KYCApprovedEvent struct {
	ApplicationID string    `json:"application_id"`
	UserID        uint64    `json:"user_id"`
	Level         KYCLevel  `json:"level"`
	ReviewerID    string    `json:"reviewer_id"`
	Timestamp     time.Time `json:"timestamp"`
}

func (e *KYCApprovedEvent) EventName() string     { return "kyc.approved" }
func (e *KYCApprovedEvent) OccurredAt() time.Time { return e.Timestamp }

// KYCRejectedEvent KYC拒绝事件
type KYCRejectedEvent struct {
	ApplicationID string    `json:"application_id"`
	UserID        uint64    `json:"user_id"`
	Reason        string    `json:"reason"`
	ReviewerID    string    `json:"reviewer_id"`
	Timestamp     time.Time `json:"timestamp"`
}

func (e *KYCRejectedEvent) EventName() string     { return "kyc.rejected" }
func (e *KYCRejectedEvent) OccurredAt() time.Time { return e.Timestamp }
