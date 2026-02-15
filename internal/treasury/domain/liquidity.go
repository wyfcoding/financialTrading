// Package domain 流动性预测领域模型
// 生成摘要：
// 1) 定义 LiquidityForecast 实体，管理未来的现金流预测
// 2) 支持按日、周、月聚合预测数据
package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ForecastType string

const (
	ForecastTypeInflow  ForecastType = "INFLOW"
	ForecastTypeOutflow ForecastType = "OUTFLOW"
)

type ConfidenceLevel string

const (
	ConfidenceLevelHigh   ConfidenceLevel = "HIGH"   // 确定性高（如已确认订单）
	ConfidenceLevelMedium ConfidenceLevel = "MEDIUM" // 可能性中（如历史规律）
	ConfidenceLevelLow    ConfidenceLevel = "LOW"    // 可能性低（如销售预测）
)

// LiquidityForecast 流动性预测条目
type LiquidityForecast struct {
	gorm.Model
	PoolID       uint64          `gorm:"column:pool_id;index;not null"`
	Date         time.Time       `gorm:"column:date;index;not null"` // 预测日期
	Type         ForecastType    `gorm:"column:type;type:varchar(32);not null"`
	Amount       decimal.Decimal `gorm:"column:amount;type:decimal(20,4);not null"`
	Currency     string          `gorm:"column:currency;type:char(3);not null"`
	Source       string          `gorm:"column:source;type:varchar(64)"` // 来源（如AP/AR系统）
	Confidence   ConfidenceLevel `gorm:"column:confidence;type:varchar(32);default:'MEDIUM'"`
	ActualAmount decimal.Decimal `gorm:"column:actual_amount;type:decimal(20,4)"` // 实际发生额（事后回填）
	Status       string          `gorm:"column:status;type:varchar(32);default:'PENDING'"`
}

func (LiquidityForecast) TableName() string { return "treasury_liquidity_forecasts" }

// LiquidityGap 流动性缺口分析结果
type LiquidityGap struct {
	Date            time.Time       `json:"date"`
	OpeningBalance  decimal.Decimal `json:"opening_balance"`
	ProjectedInflow decimal.Decimal `json:"projected_inflow"`
	ProjectedOutflow decimal.Decimal `json:"projected_outflow"`
	NetCashFlow     decimal.Decimal `json:"net_cash_flow"`
	ClosingBalance  decimal.Decimal `json:"closing_balance"`
	Gap             decimal.Decimal `json:"gap"` // 负数表示缺口
}
