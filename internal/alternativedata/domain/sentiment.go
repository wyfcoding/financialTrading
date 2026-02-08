package domain

import (
	"time"

	"gorm.io/gorm"
)

// Sentiment 另类舆情实体
type Sentiment struct {
	gorm.Model
	Symbol    string    `gorm:"column:symbol;type:varchar(20);unique_index;not null"`
	Score     float64   `gorm:"column:score;not null"`
	Trend     string    `gorm:"column:trend;type:varchar(20)"`
	Timestamp time.Time `gorm:"column:timestamp;not null"`
}

func (Sentiment) TableName() string { return "alternative_sentiments" }
