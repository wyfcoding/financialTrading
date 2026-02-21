package domain

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 错误码
var (
	ErrScheduleNotFound = errors.New("fee schedule not found")
	ErrTradeFeeNotFound = errors.New("trade fee record not found")
)

// FeeSchedule 手续费率表聚合根
type FeeSchedule struct {
	gorm.Model
	ID         string  `gorm:"type:varchar(64);uniqueIndex;not null"`
	Name       string  `gorm:"type:varchar(128);not null"`
	UserTier   string  `gorm:"type:varchar(32);index:idx_tier_asset"`
	AssetClass string  `gorm:"type:varchar(32);index:idx_tier_asset"`
	BaseRate   float64 `gorm:"type:decimal(10,6);not null"` // 例如 0.001 代表 0.1%
	MinFee     float64 `gorm:"type:decimal(20,4)"`
	MaxFee     float64 `gorm:"type:decimal(20,4)"`
}

// TradeFeeRecord 交易手续费明细
type TradeFeeRecord struct {
	gorm.Model
	TradeID      string         `gorm:"type:varchar(64);uniqueIndex;not null"`
	OrderID      string         `gorm:"type:varchar(64);index"`
	UserID       string         `gorm:"type:varchar(64);index"`
	TotalFee     float64        `gorm:"type:decimal(20,4);not null"`
	Currency     string         `gorm:"type:varchar(16);not null"`
	CalculatedAt time.Time      `gorm:"not null"`
	Components   []FeeComponent `gorm:"foreignKey:TradeFeeRecordID;references:TradeID"`
}

// FeeComponent 手续费组成部分明细 (一对多)
type FeeComponent struct {
	gorm.Model
	TradeFeeRecordID string  `gorm:"type:varchar(64);index;not null"`
	Type             int32   `gorm:"type:int;not null"` // 对应 pb.FeeType
	Amount           float64 `gorm:"type:decimal(20,4);not null"`
	Currency         string  `gorm:"type:varchar(16);not null"`
	Description      string  `gorm:"type:varchar(255)"`
}

// FeeRepository 仓储接口
type FeeRepository interface {
	SaveSchedule(ctx context.Context, s *FeeSchedule) error
	GetSchedule(ctx context.Context, id string) (*FeeSchedule, error)
	GetScheduleByTier(ctx context.Context, tier, assetClass string) (*FeeSchedule, error)
	ListSchedules(ctx context.Context, tier, assetClass string) ([]*FeeSchedule, error)

	SaveTradeFee(ctx context.Context, f *TradeFeeRecord) error
	GetTradeFees(ctx context.Context, tradeID string) (*TradeFeeRecord, error)

	CalculateRebate(ctx context.Context, userID string, startTime, endTime time.Time) (float64, error)
}

// Calculate 根据费率表计算单笔手续费
func (s *FeeSchedule) Calculate(amount float64) float64 {
	fee := amount * s.BaseRate
	if s.MinFee > 0 && fee < s.MinFee {
		fee = s.MinFee
	}
	if s.MaxFee > 0 && fee > s.MaxFee {
		fee = s.MaxFee
	}
	return fee
}
