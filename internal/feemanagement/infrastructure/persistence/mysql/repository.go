package mysql

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialTrading/internal/feemanagement/domain"
	"github.com/wyfcoding/pkg/database"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
)

// FeeScheduleModel GORM 模型.
type FeeScheduleModel struct {
	gorm.Model
	ScheduleID string  `gorm:"column:schedule_id;uniqueIndex;type:varchar(64);not null"`
	Name       string  `gorm:"column:name;type:varchar(255)"`
	UserTier   string  `gorm:"column:user_tier;index:idx_tier_asset;type:varchar(64)"`
	AssetClass string  `gorm:"column:asset_class;index:idx_tier_asset;type:varchar(64)"`
	BaseRate   float64 `gorm:"column:base_rate;type:decimal(18,8)"`
	MinFee     float64 `gorm:"column:min_fee;type:decimal(18,4)"`
	MaxFee     float64 `gorm:"column:max_fee;type:decimal(18,4)"`
}

func (FeeScheduleModel) TableName() string { return "fee_schedules" }

// TradeFeeModel 交易手续费记录.
type TradeFeeModel struct {
	gorm.Model
	TradeID    string  `gorm:"column:trade_id;uniqueIndex;type:varchar(64);not null"`
	OrderID    string  `gorm:"column:order_id;index;type:varchar(64)"`
	UserID     string  `gorm:"column:user_id;index;type:varchar(64)"`
	TotalFee   float64 `gorm:"column:total_fee;type:decimal(18,4)"`
	Currency   string  `gorm:"column:currency;type:varchar(10)"`
	Components string  `gorm:"column:components_json;type:longtext"`
}

func (TradeFeeModel) TableName() string { return "trade_fees" }

type feeRepository struct {
	db     *database.DB
	logger *logging.Logger
}

func NewFeeRepository(db *database.DB, logger *logging.Logger) domain.FeeRepository {
	return &feeRepository{db: db, logger: logger}
}

func (r *feeRepository) SaveSchedule(ctx context.Context, s *domain.FeeSchedule) error {
	m := &FeeScheduleModel{
		ScheduleID: s.ID,
		Name:       s.Name,
		UserTier:   s.UserTier,
		AssetClass: s.AssetClass,
		BaseRate:   s.BaseRate,
		MinFee:     s.MinFee,
		MaxFee:     s.MaxFee,
	}
	return r.db.RawDB().WithContext(ctx).Save(m).Error
}

func (r *feeRepository) GetScheduleByTier(ctx context.Context, tier, assetClass string) (*domain.FeeSchedule, error) {
	var m FeeScheduleModel
	if err := r.db.RawDB().WithContext(ctx).Where("user_tier = ? AND asset_class = ?", tier, assetClass).First(&m).Error; err != nil {
		return nil, err
	}
	return toScheduleDomain(&m), nil
}

func (r *feeRepository) ListSchedules(ctx context.Context) ([]*domain.FeeSchedule, error) {
	var models []*FeeScheduleModel
	if err := r.db.RawDB().WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	res := make([]*domain.FeeSchedule, len(models))
	for i, m := range models {
		res[i] = toScheduleDomain(m)
	}
	return res, nil
}

func (r *feeRepository) SaveTradeFee(ctx context.Context, f *domain.TradeFeeRecord) error {
	comps, _ := json.Marshal(f.Components)
	m := &TradeFeeModel{
		TradeID:    f.TradeID,
		OrderID:    f.OrderID,
		UserID:     f.UserID,
		TotalFee:   f.TotalFee,
		Currency:   f.Currency,
		Components: string(comps),
	}
	return r.db.RawDB().WithContext(ctx).Save(m).Error
}

func (r *feeRepository) GetTradeFees(ctx context.Context, tradeID string) (*domain.TradeFeeRecord, error) {
	var m TradeFeeModel
	if err := r.db.RawDB().WithContext(ctx).Where("trade_id = ?", tradeID).First(&m).Error; err != nil {
		return nil, err
	}
	return toTradeFeeDomain(&m), nil
}

func toScheduleDomain(m *FeeScheduleModel) *domain.FeeSchedule {
	return &domain.FeeSchedule{
		ID:         m.ScheduleID,
		Name:       m.Name,
		UserTier:   m.UserTier,
		AssetClass: m.AssetClass,
		BaseRate:   m.BaseRate,
		MinFee:     m.MinFee,
		MaxFee:     m.MaxFee,
		CreatedAt:  m.CreatedAt,
	}
}

func toTradeFeeDomain(m *TradeFeeModel) *domain.TradeFeeRecord {
	var comps []domain.FeeComponent
	_ = json.Unmarshal([]byte(m.Components), &comps)
	return &domain.TradeFeeRecord{
		TradeID:      m.TradeID,
		OrderID:      m.OrderID,
		UserID:       m.UserID,
		TotalFee:     m.TotalFee,
		Currency:     m.Currency,
		Components:   comps,
		CalculatedAt: m.CreatedAt,
	}
}
