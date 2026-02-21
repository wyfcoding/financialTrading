// Package domain 市场监察服务仓储接口
// 生成摘要：
//  1. 定义 AlertRepository / RuleRepository / UserScoreRepository / OrderEventRepository
//  2. 支持分页查询、条件过滤
//  3. 遵循 DDD 仓储接口设计，infrastructure 层实现
package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// AlertRepository 告警仓储接口
type AlertRepository interface {
	// Save 保存或更新告警
	Save(ctx context.Context, alert *SurveillanceAlert) error
	// GetByID 根据 alertID 获取告警
	GetByID(ctx context.Context, alertID string) (*SurveillanceAlert, error)
	// ListByStatus 按状态列表查询
	ListByStatus(ctx context.Context, status AlertStatus, page, pageSize int) ([]*SurveillanceAlert, int64, error)
	// ListByUserID 按用户查询告警
	ListByUserID(ctx context.Context, userID string, page, pageSize int) ([]*SurveillanceAlert, int64, error)
	// ListByFilters 综合条件查询
	ListByFilters(ctx context.Context, filters AlertFilters, page, pageSize int) ([]*SurveillanceAlert, int64, error)
}

// AlertFilters 告警查询过滤条件
type AlertFilters struct {
	Status   *AlertStatus
	Severity *AlertSeverity
	UserID   string
	Symbol   string
}

// RuleRepository 规则仓储接口
type RuleRepository interface {
	// Save 保存或更新规则
	Save(ctx context.Context, rule *SurveillanceRule) error
	// GetByID 根据 ruleID 获取规则
	GetByID(ctx context.Context, ruleID string) (*SurveillanceRule, error)
	// ListByType 按操纵类型列出规则
	ListByType(ctx context.Context, manipType ManipulationType) ([]*SurveillanceRule, error)
	// ListEnabled 列出所有启用的规则
	ListEnabled(ctx context.Context) ([]*SurveillanceRule, error)
	// ListAll 列出全部规则
	ListAll(ctx context.Context) ([]*SurveillanceRule, error)
}

// UserScoreRepository 用户评分仓储接口
type UserScoreRepository interface {
	// Save 保存或更新评分
	Save(ctx context.Context, score *UserSurveillanceScore) error
	// GetByUserID 获取用户评分
	GetByUserID(ctx context.Context, userID string) (*UserSurveillanceScore, error)
}

// OrderEventRepository 订单事件存储接口
// 说明：用于存储和检索用于检测分析的订单事件时间序列
type OrderEventRepository interface {
	// SaveEvent 存储单个事件
	SaveEvent(ctx context.Context, event *OrderEventRecord) error
	// SaveBatch 批量存储事件
	SaveBatch(ctx context.Context, events []*OrderEventRecord) error
	// GetByUserAndSymbol 获取指定用户和标的在时间窗口内的事件
	GetByUserAndSymbol(ctx context.Context, userID, symbol string, start, end interface{}) ([]*OrderEventRecord, error)
	// GetByUser 获取指定用户在时间窗口内的全部事件
	GetByUser(ctx context.Context, userID string, start, end interface{}) ([]*OrderEventRecord, error)
}

// OrderEventRecord 订单事件持久化记录
type OrderEventRecord struct {
	gorm.Model
	// 订单ID
	OrderID string `gorm:"column:order_id;type:varchar(64);index;not null" json:"order_id"`
	// 用户ID
	UserID string `gorm:"column:user_id;type:varchar(64);index;not null" json:"user_id"`
	// 标的代码
	Symbol string `gorm:"column:symbol;type:varchar(32);index;not null" json:"symbol"`
	// 事件类型：PLACE / CANCEL / FILL / MODIFY
	EventType OrderEventType `gorm:"column:event_type;type:tinyint;not null" json:"event_type"`
	// 买卖方向
	Side string `gorm:"column:side;type:varchar(4);not null" json:"side"`
	// 价格
	Price string `gorm:"column:price;type:varchar(32)" json:"price"`
	// 数量
	Quantity string `gorm:"column:quantity;type:varchar(32)" json:"quantity"`
	// 交易场所
	Venue string `gorm:"column:venue;type:varchar(32)" json:"venue"`
	// 账户ID
	AccountID string `gorm:"column:account_id;type:varchar(64)" json:"account_id"`
	// 事件发生时间
	EventTime time.Time `gorm:"column:event_time;type:datetime(6);index;not null" json:"event_time"`
}
