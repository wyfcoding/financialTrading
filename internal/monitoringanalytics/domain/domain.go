// 包 监控分析服务的领域模型
package domain

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Metric 指标实体
// 代表一个时间点上的监控指标数据
type Metric struct {
	gorm.Model
	// Name 指标名称
	Name string `gorm:"column:name;type:varchar(100);index;not null"`
	// Value 指标值
	Value decimal.Decimal `gorm:"column:value;type:decimal(32,18);not null"`
	// Tags 标签 (内存中 Map 表示)
	Tags map[string]string `gorm:"-"`
	// TagsJSON 标签 (数据库存储 JSON 字符串)
	TagsJSON string `gorm:"column:tags;type:text"`
	// Timestamp 时间戳
	Timestamp int64 `gorm:"column:timestamp;type:bigint;index;not null"`
}

// SystemHealth 系统健康状态实体
// 记录各个微服务的健康检查结果
type SystemHealth struct {
	gorm.Model
	// ServiceName 服务名称
	ServiceName string `gorm:"column:service_name;type:varchar(100);index;not null"`
	// Status 状态: UP(正常), DOWN(停止), DEGRADED(降级)
	Status string `gorm:"column:status;type:varchar(20);not null"`
	// Message 详细消息
	Message string `gorm:"column:message;type:text"`
	// LastChecked 上次检查时间
	LastChecked int64 `gorm:"column:last_checked;type:bigint;not null"`
}

// End of domain file
