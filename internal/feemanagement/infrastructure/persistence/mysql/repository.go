package mysql

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/feemanagement/domain"
	"gorm.io/gorm"
)

type feeRepository struct {
	db *gorm.DB
}

// NewFeeRepository 初始化基于 MySQL 的费用仓储
func NewFeeRepository(db *gorm.DB) domain.FeeRepository {
	_ = db.AutoMigrate(&domain.FeeSchedule{}, &domain.TradeFeeRecord{}, &domain.FeeComponent{})
	return &feeRepository{db: db}
}

func (r *feeRepository) SaveSchedule(ctx context.Context, s *domain.FeeSchedule) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *feeRepository) GetSchedule(ctx context.Context, id string) (*domain.FeeSchedule, error) {
	var sched domain.FeeSchedule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&sched).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrScheduleNotFound
		}
		return nil, err
	}
	return &sched, nil
}

func (r *feeRepository) GetScheduleByTier(ctx context.Context, tier, assetClass string) (*domain.FeeSchedule, error) {
	var sched domain.FeeSchedule
	err := r.db.WithContext(ctx).
		Where("user_tier = ? AND asset_class = ?", tier, assetClass).
		Order("created_at DESC").
		First(&sched).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrScheduleNotFound
		}
		return nil, err
	}
	return &sched, nil
}

func (r *feeRepository) ListSchedules(ctx context.Context, tier, assetClass string) ([]*domain.FeeSchedule, error) {
	var scheds []*domain.FeeSchedule
	query := r.db.WithContext(ctx)

	if tier != "" {
		query = query.Where("user_tier = ?", tier)
	}
	if assetClass != "" {
		query = query.Where("asset_class = ?", assetClass)
	}

	if err := query.Order("created_at DESC").Find(&scheds).Error; err != nil {
		return nil, err
	}
	return scheds, nil
}

func (r *feeRepository) SaveTradeFee(ctx context.Context, f *domain.TradeFeeRecord) error {
	// 使用事务保存交易记录及费用明细组合
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(f).Error; err != nil {
			return err
		}
		for _, comp := range f.Components {
			comp.TradeFeeRecordID = f.TradeID // 保证外键一致
			if err := tx.Create(&comp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *feeRepository) GetTradeFees(ctx context.Context, tradeID string) (*domain.TradeFeeRecord, error) {
	var record domain.TradeFeeRecord
	if err := r.db.WithContext(ctx).Preload("Components").Where("trade_id = ?", tradeID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTradeFeeNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r *feeRepository) CalculateRebate(ctx context.Context, userID string, startTime, endTime time.Time) (float64, error) {
	var totalFee float64

	// 从 TradeFeeRecord 聚合计算总产生的手续费，返佣比例这里简化为 10%
	err := r.db.WithContext(ctx).
		Model(&domain.TradeFeeRecord{}).
		Where("user_id = ? AND calculated_at >= ? AND calculated_at <= ?", userID, startTime, endTime).
		Select("COALESCE(SUM(total_fee), 0)").
		Scan(&totalFee).Error

	if err != nil {
		return 0, err
	}

	// 简单的 10% 返佣逻辑 (后续可基于更复杂的返佣规则表)
	rebateAmount := totalFee * 0.10
	return rebateAmount, nil
}
