// 生成摘要：从 benchmark 合并到 portfolio 域。基准指数属组合管理子能力。
package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BenchmarkIndex 基准指数实体。
// 用于量化交易中的 Alpha/Beta 收益率对比，如沪深300、S&P 500实时点位。
type BenchmarkIndex struct {
	gorm.Model
	// Code 指数代码，如 SPX。
	Code string `gorm:"type:varchar(32);uniqueIndex" json:"code"`
	// Name 指数名称。
	Name string `gorm:"type:varchar(64)" json:"name"`
	// Publisher 发布机构。
	Publisher string `gorm:"type:varchar(64)" json:"publisher"`
	// BaseValue 基准值。
	BaseValue decimal.Decimal `gorm:"type:decimal(20,4)" json:"base_value"`
	// LaunchDate 发布日期。
	LaunchDate time.Time `json:"launch_date"`
}

// IndexConstituent 指数成分股结构（权重图）。
type IndexConstituent struct {
	gorm.Model
	// IndexID 关联指数 ID。
	IndexID uint `gorm:"index" json:"index_id"`
	// Symbol 成分股代码。
	Symbol string `gorm:"type:varchar(32)" json:"symbol"`
	// Weight 成分权重比例。
	Weight decimal.Decimal `gorm:"type:decimal(10,8);comment:成分权重比例" json:"weight"`
}
