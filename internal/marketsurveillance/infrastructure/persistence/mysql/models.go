// Package mysql 市场监察服务 GORM 模型定义
// 生成摘要：
//  1. 数据库表自动迁移辅助函数
//  2. 域模型直接作为 GORM 模型使用（在 domain 层定义了 gorm 标签）
package mysql

import (
	"gorm.io/gorm"

	"github.com/wyfcoding/financialtrading/internal/marketsurveillance/domain"
)

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.SurveillanceAlert{},
		&domain.SurveillanceRule{},
		&domain.UserSurveillanceScore{},
		&domain.OrderEventRecord{},
	)
}
