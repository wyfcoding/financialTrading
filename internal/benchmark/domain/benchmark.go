// Package domain 基准指数领域模型
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BenchmarkIndex 基准指数
type BenchmarkIndex struct {
	gorm.Model
	Symbol      string `gorm:"column:symbol;type:varchar(32);uniqueIndex;not null"`
	Name        string `gorm:"column:name;type:varchar(128);not null"`
	Currency    string `gorm:"column:currency;type:char(3);not null"`
	Description string `gorm:"column:description;type:text"`
	Source      string `gorm:"column:source;type:varchar(32)"` // 来源: BLOOMBERG, REUTERS
}

func (BenchmarkIndex) TableName() string { return "bm_indices" }

// IndexConstituent 指数成分股
type IndexConstituent struct {
	gorm.Model
	IndexSymbol string          `gorm:"column:index_symbol;index;not null"`
	StockSymbol string          `gorm:"column:stock_symbol;index;not null"`
	Weight      decimal.Decimal `gorm:"column:weight;type:decimal(10,8);not null"` // 权重
	EffectiveDate time.Time     `gorm:"column:effective_date;index;not null"`      // 生效日期
}

func (IndexConstituent) TableName() string { return "bm_constituents" }

type BenchmarkRepository interface {
	SaveIndex(ctx context.Context, idx *BenchmarkIndex) error
	GetIndex(ctx context.Context, symbol string) (*BenchmarkIndex, error)
	
	SaveConstituents(ctx context.Context, constituents []IndexConstituent) error
	GetConstituents(ctx context.Context, indexSymbol string, date time.Time) ([]IndexConstituent, error)
}
