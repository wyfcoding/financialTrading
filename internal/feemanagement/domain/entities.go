package domain

import (
	"context"
	"time"

	pb "github.com/wyfcoding/financialTrading/go-api/feemanagement/v1"
)

// FeeSchedule 聚合根，代表手续费率表.
type FeeSchedule struct {
	ID         string
	Name       string
	UserTier   string
	AssetClass string
	BaseRate   float64 // 例如 0.001 代表 0.1%
	MinFee     float64
	MaxFee     float64
	CreatedAt  time.Time
}

// TradeFeeRecord 代表一笔交易的实际手续费明细.
type TradeFeeRecord struct {
	TradeID     string
	OrderID     string
	UserID      string
	TotalFee    float64
	Currency    string
	Components  []FeeComponent
	CalculatedAt time.Time
}

// FeeComponent 手续费组成部分.
type FeeComponent struct {
	Type        pb.FeeType
	Amount      float64
	Currency    string
	Description string
}

// FeeRepository 仓储接口.
type FeeRepository interface {
	SaveSchedule(ctx context.Context, s *FeeSchedule) error
	GetScheduleByTier(ctx context.Context, tier, assetClass string) (*FeeSchedule, error)
	ListSchedules(ctx context.Context) ([]*FeeSchedule, error)
	SaveTradeFee(ctx context.Context, f *TradeFeeRecord) error
	GetTradeFees(ctx context.Context, tradeID string) (*TradeFeeRecord, error)
}

// 领域辅助方法：根据成交信息计算手续费
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
创新：通过领域模型直接封装计算逻辑，保证了业务规则的内聚。
