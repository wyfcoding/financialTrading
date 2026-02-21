package domain

import (
	"time"

	"gorm.io/gorm"
)

type ReplayStatus string

const (
	ReplayRunning  ReplayStatus = "RUNNING"
	ReplayPaused   ReplayStatus = "PAUSED"
	ReplayFinished ReplayStatus = "FINISHED"
)

// ReplayTask 回放任务聚合。
type ReplayTask struct {
	gorm.Model
	Symbol      string       `gorm:"column:symbol;not null"`
	StartTime   time.Time    `gorm:"column:start_time"`
	EndTime     time.Time    `gorm:"column:end_time"`
	SpeedFactor float64      `gorm:"column:speed_factor;default:1.0"`
	Status      ReplayStatus `gorm:"column:status;default:'RUNNING'"`
	Topic       string       `gorm:"column:target_topic"`
}

type MarketTick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}
