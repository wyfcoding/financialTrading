// Package domain 市场监察服务领域事件定义
// 生成摘要：
//  1. 定义告警生命周期事件（创建/调查/升级/关闭）
//  2. 定义用户评分更新事件
//  3. 所有事件实现 DomainEvent 接口
package domain

import "time"

// DomainEvent 领域事件接口
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// AlertCreatedEvent 告警创建事件
type AlertCreatedEvent struct {
	AlertID         string           `json:"alert_id"`
	UserID          string           `json:"user_id"`
	Symbol          string           `json:"symbol"`
	Type            ManipulationType `json:"type"`
	Severity        AlertSeverity    `json:"severity"`
	ConfidenceScore float64          `json:"confidence_score"`
	Timestamp       time.Time        `json:"timestamp"`
}

// EventName 事件名
func (e *AlertCreatedEvent) EventName() string { return "surveillance.alert.created" }

// OccurredAt 发生时间
func (e *AlertCreatedEvent) OccurredAt() time.Time { return e.Timestamp }

// AlertInvestigationStartedEvent 告警调查开始事件
type AlertInvestigationStartedEvent struct {
	AlertID    string    `json:"alert_id"`
	ReviewerID string    `json:"reviewer_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// EventName 事件名
func (e *AlertInvestigationStartedEvent) EventName() string {
	return "surveillance.alert.investigation_started"
}

// OccurredAt 发生时间
func (e *AlertInvestigationStartedEvent) OccurredAt() time.Time { return e.Timestamp }

// AlertEscalatedEvent 告警升级事件
type AlertEscalatedEvent struct {
	AlertID   string    `json:"alert_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// EventName 事件名
func (e *AlertEscalatedEvent) EventName() string { return "surveillance.alert.escalated" }

// OccurredAt 发生时间
func (e *AlertEscalatedEvent) OccurredAt() time.Time { return e.Timestamp }

// AlertClosedEvent 告警关闭事件
type AlertClosedEvent struct {
	AlertID   string           `json:"alert_id"`
	Confirmed bool             `json:"confirmed"`
	UserID    string           `json:"user_id"`
	Type      ManipulationType `json:"type"`
	Timestamp time.Time        `json:"timestamp"`
}

// EventName 事件名
func (e *AlertClosedEvent) EventName() string { return "surveillance.alert.closed" }

// OccurredAt 发生时间
func (e *AlertClosedEvent) OccurredAt() time.Time { return e.Timestamp }

// UserScoreUpdatedEvent 用户评分更新事件
type UserScoreUpdatedEvent struct {
	UserID       string    `json:"user_id"`
	OverallScore float64   `json:"overall_score"`
	Timestamp    time.Time `json:"timestamp"`
}

// EventName 事件名
func (e *UserScoreUpdatedEvent) EventName() string { return "surveillance.user_score.updated" }

// OccurredAt 发生时间
func (e *UserScoreUpdatedEvent) OccurredAt() time.Time { return e.Timestamp }
