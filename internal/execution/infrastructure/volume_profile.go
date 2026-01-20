package infrastructure

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type MockVolumeProfileProvider struct{}

func NewMockVolumeProfileProvider() *MockVolumeProfileProvider {
	return &MockVolumeProfileProvider{}
}

// GetProfile 返回一个模拟的日内成交量曲线 (典型的 U 型曲线)
func (p *MockVolumeProfileProvider) GetProfile(symbol string) ([]domain.VolumeProfileItem, error) {
	// 简单的 U 型分布模拟 (9:30 - 16:00)
	profile := []domain.VolumeProfileItem{
		{TimeSlot: "09:30", Ratio: decimal.NewFromFloat(0.15)},
		{TimeSlot: "10:00", Ratio: decimal.NewFromFloat(0.10)},
		{TimeSlot: "11:00", Ratio: decimal.NewFromFloat(0.05)},
		{TimeSlot: "12:00", Ratio: decimal.NewFromFloat(0.03)},
		{TimeSlot: "13:00", Ratio: decimal.NewFromFloat(0.02)},
		{TimeSlot: "14:00", Ratio: decimal.NewFromFloat(0.05)},
		{TimeSlot: "15:00", Ratio: decimal.NewFromFloat(0.20)},
		{TimeSlot: "15:30", Ratio: decimal.NewFromFloat(0.40)},
	}
	return profile, nil
}
