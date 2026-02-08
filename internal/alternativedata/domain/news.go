package domain

import (
	"time"

	"gorm.io/gorm"
)

// NewsItem 另类新闻实体
type NewsItem struct {
	gorm.Model
	NewsID         string    `gorm:"column:news_id;type:varchar(32);unique_index;not null"`
	Symbol         string    `gorm:"column:symbol;type:varchar(20);index;not null"`
	Title          string    `gorm:"column:title;type:varchar(255);not null"`
	Content        string    `gorm:"column:content;type:text"`
	Source         string    `gorm:"column:source;type:varchar(50)"`
	SentimentScore float64   `gorm:"column:sentiment_score"`
	PublishedAt    time.Time `gorm:"column:published_at"`
}

func (NewsItem) TableName() string { return "alternative_news" }
