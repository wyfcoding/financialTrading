// 包 风险管理服务的领域模型、实体、聚合、值对象、领域服务、仓储接口
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// RiskAssessment 风险评估实体
// 代表对用户交易风险的评估结果
type RiskAssessment struct {
	gorm.Model
	// 评估 ID
	AssessmentID string `gorm:"column:assessment_id;type:varchar(50);uniqueIndex;not null" json:"assessment_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);not null" json:"symbol"`
	// 买卖方向
	Side string `gorm:"column:side;type:varchar(10);not null" json:"side"`
	// 数量
	Quantity decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 价格
	Price decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// 风险等级
	RiskLevel RiskLevel `gorm:"column:risk_level;type:varchar(20);not null" json:"risk_level"`
	// 风险分数（0-100）
	RiskScore decimal.Decimal `gorm:"column:risk_score;type:decimal(5,2);not null" json:"risk_score"`
	// 保证金要求
	MarginRequirement decimal.Decimal `gorm:"column:margin_requirement;type:decimal(20,8);not null" json:"margin_requirement"`
	// 是否允许交易
	IsAllowed bool `gorm:"column:is_allowed;type:boolean;not null" json:"is_allowed"`
	// 原因
	Reason string `gorm:"column:reason;type:text" json:"reason"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
}

// RiskMetrics 风险指标实体
// 代表用户的风险指标汇总
type RiskMetrics struct {
	gorm.Model
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);uniqueIndex;not null" json:"user_id"`
	// 在 95% 置信水平下的风险价值 (Value at Risk)
	VaR95 decimal.Decimal `gorm:"column:var_95;type:decimal(20,8);not null" json:"var_95"`
	// 在 99% 置信水平下的风险价值 (Value at Risk)
	VaR99 decimal.Decimal `gorm:"column:var_99;type:decimal(20,8);not null" json:"var_99"`
	// 最大回撤
	MaxDrawdown decimal.Decimal `gorm:"column:max_drawdown;type:decimal(20,8);not null" json:"max_drawdown"`
	// 夏普比率
	SharpeRatio decimal.Decimal `gorm:"column:sharpe_ratio;type:decimal(20,8);not null" json:"sharpe_ratio"`
	// 相关系数
	Correlation decimal.Decimal `gorm:"column:correlation;type:decimal(20,8);not null" json:"correlation"`
	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null" json:"updated_at"`
}

// RiskLimit 风险限额实体
// 代表用户的风险限额配置
type RiskLimit struct {
	gorm.Model
	// 限额 ID
	LimitID string `gorm:"column:limit_id;type:varchar(50);uniqueIndex;not null" json:"limit_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 限额类型（POSITION_SIZE, DAILY_LOSS, LEVERAGE）
	LimitType string `gorm:"column:limit_type;type:varchar(50);not null" json:"limit_type"`
	// 限额值
	LimitValue decimal.Decimal `gorm:"column:limit_value;type:decimal(20,8);not null" json:"limit_value"`
	// 当前值
	CurrentValue decimal.Decimal `gorm:"column:current_value;type:decimal(20,8);not null" json:"current_value"`
	// 是否超限
	IsExceeded bool `gorm:"column:is_exceeded;type:boolean;not null" json:"is_exceeded"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null" json:"updated_at"`
}

// RiskAlert 风险告警实体
// 代表风险告警信息
type RiskAlert struct {
	gorm.Model
	// 告警 ID
	AlertID string `gorm:"column:alert_id;type:varchar(50);uniqueIndex;not null" json:"alert_id"`
	// 用户 ID
	UserID string `gorm:"column:user_id;type:varchar(50);index;not null" json:"user_id"`
	// 告警类型
	AlertType string `gorm:"column:alert_type;type:varchar(50);not null" json:"alert_type"`
	// 严重程度
	Severity string `gorm:"column:severity;type:varchar(20);not null" json:"severity"`
	// 消息
	Message string `gorm:"column:message;type:text;not null" json:"message"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
}

// RiskAssessmentRepository 风险评估仓储接口
type RiskAssessmentRepository interface {
	// 保存风险评估
	Save(ctx context.Context, assessment *RiskAssessment) error
	// 获取风险评估
	Get(ctx context.Context, assessmentID string) (*RiskAssessment, error)
	// 获取用户最新评估
	GetLatestByUser(ctx context.Context, userID string) (*RiskAssessment, error)
}

// RiskMetricsRepository 风险指标仓储接口
type RiskMetricsRepository interface {
	// 保存风险指标
	Save(ctx context.Context, metrics *RiskMetrics) error
	// 获取风险指标
	Get(ctx context.Context, userID string) (*RiskMetrics, error)
	// 更新风险指标
	Update(ctx context.Context, metrics *RiskMetrics) error
}

// RiskLimitRepository 风险限额仓储接口
type RiskLimitRepository interface {
	// 保存风险限额
	Save(ctx context.Context, limit *RiskLimit) error
	// 获取风险限额
	Get(ctx context.Context, limitID string) (*RiskLimit, error)
	// 获取用户限额
	GetByUser(ctx context.Context, userID string, limitType string) (*RiskLimit, error)
	// 更新风险限额
	Update(ctx context.Context, limit *RiskLimit) error
}

// RiskAlertRepository 风险告警仓储接口
type RiskAlertRepository interface {
	// 保存风险告警
	Save(ctx context.Context, alert *RiskAlert) error
	// 获取用户告警
	GetByUser(ctx context.Context, userID string, limit int) ([]*RiskAlert, error)
	// 删除已读告警
	DeleteRead(ctx context.Context, alertID string) error
}

// RiskDomainService 风险领域服务
// 提供风险相关的业务逻辑
type RiskDomainService interface {
	// 评估交易风险
	AssessTradeRisk(ctx context.Context, userID, symbol, side string, quantity, price decimal.Decimal) (*RiskAssessment, error)
	// 计算风险指标
	CalculateRiskMetrics(ctx context.Context, userID string) (*RiskMetrics, error)
	// 检查风险限额
	CheckRiskLimit(ctx context.Context, userID, limitType string) (*RiskLimit, error)
	// 生成风险告警
	GenerateRiskAlert(ctx context.Context, userID, alertType, severity, message string) (*RiskAlert, error)
}
