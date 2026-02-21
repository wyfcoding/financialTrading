// Package domain 市场监察服务领域层 - 实体与聚合
// 生成摘要：
//  1. 定义 Alert 聚合根、Rule 实体、UserScore 值对象
//  2. 状态机驱动的告警生命周期（Open → Investigating → Escalated → Closed）
//  3. 包含领域事件收集能力
package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ManipulationType 市场操纵类型
type ManipulationType int

const (
	ManipulationUnspecified  ManipulationType = 0
	ManipulationSpoofing     ManipulationType = 1
	ManipulationLayering     ManipulationType = 2
	ManipulationWashTrading  ManipulationType = 3
	ManipulationFrontRun     ManipulationType = 4
	ManipulationClosingPrice ManipulationType = 5
	ManipulationPumpAndDump  ManipulationType = 6
)

// AlertSeverity 告警严重程度
type AlertSeverity int

const (
	SeverityLow      AlertSeverity = 1
	SeverityMedium   AlertSeverity = 2
	SeverityHigh     AlertSeverity = 3
	SeverityCritical AlertSeverity = 4
)

// AlertStatus 告警状态
type AlertStatus int

const (
	AlertOpen            AlertStatus = 1
	AlertInvestigating   AlertStatus = 2
	AlertEscalated       AlertStatus = 3
	AlertClosedConfirmed AlertStatus = 4
	AlertClosedFalse     AlertStatus = 5
)

// SurveillanceAlert 监察告警聚合根
// 说明：告警由检测引擎自动创建，经人工审核后关闭
type SurveillanceAlert struct {
	gorm.Model
	// 告警唯一标识
	AlertID string `gorm:"column:alert_id;type:varchar(64);uniqueIndex;not null" json:"alert_id"`
	// 操纵类型
	Type ManipulationType `gorm:"column:type;type:tinyint;not null;index" json:"type"`
	// 严重程度
	Severity AlertSeverity `gorm:"column:severity;type:tinyint;not null;index" json:"severity"`
	// 当前状态
	Status AlertStatus `gorm:"column:status;type:tinyint;not null;index" json:"status"`
	// 涉事用户ID
	UserID string `gorm:"column:user_id;type:varchar(64);not null;index" json:"user_id"`
	// 涉事标的
	Symbol string `gorm:"column:symbol;type:varchar(32);not null;index" json:"symbol"`
	// 告警描述
	Description string `gorm:"column:description;type:text" json:"description"`
	// 置信度评分 (0.0~1.0)
	ConfidenceScore float64 `gorm:"column:confidence_score;type:decimal(5,4)" json:"confidence_score"`
	// 关联规则ID
	RuleID string `gorm:"column:rule_id;type:varchar(64);index" json:"rule_id"`
	// 关联订单ID列表（JSON 数组存储）
	RelatedOrderIDs string `gorm:"column:related_order_ids;type:text" json:"related_order_ids"`
	// 审核人
	ReviewerID string `gorm:"column:reviewer_id;type:varchar(64)" json:"reviewer_id"`
	// 审核评论
	ReviewComment string `gorm:"column:review_comment;type:text" json:"review_comment"`
	// 检测时间
	DetectedAt time.Time `gorm:"column:detected_at;not null" json:"detected_at"`
	// 审核时间
	ReviewedAt *time.Time `gorm:"column:reviewed_at" json:"reviewed_at"`
	// 领域事件
	domainEvents []DomainEvent `gorm:"-"`
}

// NewAlert 创建新告警
func NewAlert(
	alertID string,
	manipType ManipulationType,
	severity AlertSeverity,
	userID, symbol, description, ruleID string,
	confidence float64,
	relatedOrders string,
) *SurveillanceAlert {
	return &SurveillanceAlert{
		AlertID:         alertID,
		Type:            manipType,
		Severity:        severity,
		Status:          AlertOpen,
		UserID:          userID,
		Symbol:          symbol,
		Description:     description,
		ConfidenceScore: confidence,
		RuleID:          ruleID,
		RelatedOrderIDs: relatedOrders,
		DetectedAt:      time.Now(),
	}
}

// StartInvestigation 开始调查
func (a *SurveillanceAlert) StartInvestigation(reviewerID string) error {
	if a.Status != AlertOpen {
		return fmt.Errorf("alert %s is not open, current status: %d", a.AlertID, a.Status)
	}
	a.Status = AlertInvestigating
	a.ReviewerID = reviewerID
	a.domainEvents = append(a.domainEvents, &AlertInvestigationStartedEvent{
		AlertID:    a.AlertID,
		ReviewerID: reviewerID,
		Timestamp:  time.Now(),
	})
	return nil
}

// Escalate 升级告警
func (a *SurveillanceAlert) Escalate(reason string) error {
	if a.Status != AlertInvestigating {
		return fmt.Errorf("alert %s must be investigating to escalate", a.AlertID)
	}
	a.Status = AlertEscalated
	a.ReviewComment = reason
	a.domainEvents = append(a.domainEvents, &AlertEscalatedEvent{
		AlertID:   a.AlertID,
		Reason:    reason,
		Timestamp: time.Now(),
	})
	return nil
}

// CloseConfirmed 确认关闭（确认操纵行为）
func (a *SurveillanceAlert) CloseConfirmed(reviewerID, comment string) error {
	if a.Status != AlertInvestigating && a.Status != AlertEscalated {
		return fmt.Errorf("alert %s cannot be closed from status %d", a.AlertID, a.Status)
	}
	now := time.Now()
	a.Status = AlertClosedConfirmed
	a.ReviewerID = reviewerID
	a.ReviewComment = comment
	a.ReviewedAt = &now
	a.domainEvents = append(a.domainEvents, &AlertClosedEvent{
		AlertID:   a.AlertID,
		Confirmed: true,
		UserID:    a.UserID,
		Type:      a.Type,
		Timestamp: now,
	})
	return nil
}

// CloseFalsePositive 误报关闭
func (a *SurveillanceAlert) CloseFalsePositive(reviewerID, comment string) error {
	if a.Status != AlertInvestigating && a.Status != AlertEscalated {
		return fmt.Errorf("alert %s cannot be closed from status %d", a.AlertID, a.Status)
	}
	now := time.Now()
	a.Status = AlertClosedFalse
	a.ReviewerID = reviewerID
	a.ReviewComment = comment
	a.ReviewedAt = &now
	a.domainEvents = append(a.domainEvents, &AlertClosedEvent{
		AlertID:   a.AlertID,
		Confirmed: false,
		UserID:    a.UserID,
		Type:      a.Type,
		Timestamp: now,
	})
	return nil
}

// GetDomainEvents 获取领域事件
func (a *SurveillanceAlert) GetDomainEvents() []DomainEvent { return a.domainEvents }

// ClearDomainEvents 清空领域事件
func (a *SurveillanceAlert) ClearDomainEvents() { a.domainEvents = nil }

// SurveillanceRule 监察规则实体
// 说明：定义检测引擎使用的阈值与窗口参数
type SurveillanceRule struct {
	gorm.Model
	// 规则唯一标识
	RuleID string `gorm:"column:rule_id;type:varchar(64);uniqueIndex;not null" json:"rule_id"`
	// 规则名称
	Name string `gorm:"column:name;type:varchar(128);not null" json:"name"`
	// 操纵类型
	Type ManipulationType `gorm:"column:type;type:tinyint;not null;index" json:"type"`
	// 是否启用
	Enabled bool `gorm:"column:enabled;default:true" json:"enabled"`
	// 检测时间窗口（秒）
	WindowSeconds int64 `gorm:"column:window_seconds;not null" json:"window_seconds"`
	// 通用阈值（触发分数）
	Threshold decimal.Decimal `gorm:"column:threshold;type:decimal(10,4)" json:"threshold"`
	// 最低撤单比率(Spoofing/Layering)
	MinCancelRatio decimal.Decimal `gorm:"column:min_cancel_ratio;type:decimal(5,4)" json:"min_cancel_ratio"`
	// 最大洗售成交比率
	MaxWashVolumeRatio decimal.Decimal `gorm:"column:max_wash_volume_ratio;type:decimal(5,4)" json:"max_wash_volume_ratio"`
	// 价格偏差阈值
	PriceDeviationThreshold decimal.Decimal `gorm:"column:price_deviation_threshold;type:decimal(10,4)" json:"price_deviation_threshold"`
	// 规则描述
	Description string `gorm:"column:description;type:text" json:"description"`
}

// UserSurveillanceScore 用户监察评分
// 说明：聚合用户在各类操纵维度的风险评分
type UserSurveillanceScore struct {
	gorm.Model
	// 用户ID
	UserID string `gorm:"column:user_id;type:varchar(64);uniqueIndex;not null" json:"user_id"`
	// 综合评分 (0.0~1.0，越高越可疑)
	OverallScore float64 `gorm:"column:overall_score;type:decimal(5,4)" json:"overall_score"`
	// 幌骗评分
	SpoofingScore float64 `gorm:"column:spoofing_score;type:decimal(5,4)" json:"spoofing_score"`
	// 洗售评分
	WashTradingScore float64 `gorm:"column:wash_trading_score;type:decimal(5,4)" json:"wash_trading_score"`
	// 分层评分
	LayeringScore float64 `gorm:"column:layering_score;type:decimal(5,4)" json:"layering_score"`
	// 总告警数
	TotalAlerts int64 `gorm:"column:total_alerts" json:"total_alerts"`
	// 已确认告警数
	ConfirmedAlerts int64 `gorm:"column:confirmed_alerts" json:"confirmed_alerts"`
}

// OrderEventType 订单事件类型
type OrderEventType int

const (
	EventPlace  OrderEventType = 1
	EventCancel OrderEventType = 2
	EventFill   OrderEventType = 3
	EventModify OrderEventType = 4
)

// OrderEvent 订单事件（值对象，用于检测引擎输入）
type OrderEvent struct {
	OrderID   string
	UserID    string
	Symbol    string
	EventType OrderEventType
	Side      string // BUY / SELL
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Timestamp time.Time
	Venue     string
	AccountID string
}

// TradingPattern 交易模式分析结果（值对象）
type TradingPattern struct {
	UserID             string
	Symbol             string
	CancelRatio        float64
	AvgOrderLifetimeMs float64
	SelfTradeRatio     float64
	TotalOrders        int64
	TotalCancels       int64
	TotalFills         int64
	RiskLevel          string
}
