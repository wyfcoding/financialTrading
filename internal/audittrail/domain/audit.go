// 变更说明：定义全局审计日志结构。
// 满足金融系统不可篡改、全链路追踪的要求。
package domain

import (
	"time"
)

type AuditLog struct {
	ID            string    `json:"id"`
	TraceID       string    `json:"trace_id"`
	UserID        string    `json:"user_id"`
	Service       string    `json:"service"`
	Action        string    `json:"action"` // create_order, login, etc
	Resource      string    `json:"resource"`
	ResourceID    string    `json:"resource_id"`
	IP            string    `json:"ip"`
	UserAgent     string    `json:"user_agent"`
	Request       string    `json:"request"`  // 脱敏后的 JSON
	Response      string    `json:"response"` // 脱敏后的 JSON
	Status        string    `json:"status"`   // SUCCESS, FAIL
	ErrorMessage  string    `json:"error_message,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type AuditRepository interface {
	Store(log *AuditLog) error
	Query(filter map[string]interface{}) ([]*AuditLog, error)
}
