// Package mysql 提供了清算仓储接口的 MySQL GORM 实现。
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettlementModel 是清算记录的数据库模型。
type SettlementModel struct {
	gorm.Model
	SettlementID   string    `gorm:"column:settlement_id;type:varchar(32);uniqueIndex;not null"`
	TradeID        string    `gorm:"column:trade_id;type:varchar(32);index;not null"`
	BuyUserID      string    `gorm:"column:buy_user_id;type:varchar(32);index;not null"`
	SellUserID     string    `gorm:"column:sell_user_id;type:varchar(32);index;not null"`
	Symbol         string    `gorm:"column:symbol;type:varchar(20);not null"`
	Quantity       string    `gorm:"column:quantity;type:decimal(32,18);not null"`
	Price          string    `gorm:"column:price;type:decimal(32,18);not null"`
	Status         string    `gorm:"column:status;type:varchar(20);index;not null"`
	SettlementTime time.Time `gorm:"column:settlement_time;type:datetime;not null"`
}

// TableName 指定表名
func (SettlementModel) TableName() string {
	return "settlements"
}

// settlementRepositoryImpl 是 domain.SettlementRepository 接口的 GORM 实现。
type settlementRepositoryImpl struct {
	db *gorm.DB
}

// NewSettlementRepository 创建清算仓储实例
func NewSettlementRepository(db *gorm.DB) domain.SettlementRepository {
	return &settlementRepositoryImpl{db: db}
}

// Save 实现 domain.SettlementRepository.Save
func (r *settlementRepositoryImpl) Save(ctx context.Context, s *domain.Settlement) error {
	model := &SettlementModel{
		Model:          s.Model,
		SettlementID:   s.SettlementID,
		TradeID:        s.TradeID,
		BuyUserID:      s.BuyUserID,
		SellUserID:     s.SellUserID,
		Symbol:         s.Symbol,
		Quantity:       s.Quantity.String(),
		Price:          s.Price.String(),
		Status:         s.Status,
		SettlementTime: s.SettlementTime,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "settlement_id"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "settlement_repository.Save failed", "settlement_id", s.SettlementID, "error", err)
		return fmt.Errorf("failed to save settlement: %w", err)
	}
	s.Model = model.Model
	return nil
}

// Get 实现 domain.SettlementRepository.Get
func (r *settlementRepositoryImpl) Get(ctx context.Context, settlementID string) (*domain.Settlement, error) {
	var model SettlementModel
	if err := r.db.WithContext(ctx).Where("settlement_id = ?", settlementID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "settlement_repository.Get failed", "settlement_id", settlementID, "error", err)
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}
	return r.toDomain(&model), nil
}

// GetByUser 实现 domain.SettlementRepository.GetByUser
func (r *settlementRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Settlement, int64, error) {
	var models []SettlementModel
	var total int64
	db := r.db.WithContext(ctx).Model(&SettlementModel{}).Where("buy_user_id = ? OR sell_user_id = ?", userID, userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "settlement_repository.GetByUser failed", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to get settlements by user: %w", err)
	}
	res := make([]*domain.Settlement, len(models))
	for i, m := range models {
		res[i] = r.toDomain(&m)
	}
	return res, total, nil
}

// GetByTrade 实现 domain.SettlementRepository.GetByTrade
func (r *settlementRepositoryImpl) GetByTrade(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var model SettlementModel
	if err := r.db.WithContext(ctx).Where("trade_id = ?", tradeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "settlement_repository.GetByTrade failed", "trade_id", tradeID, "error", err)
		return nil, fmt.Errorf("failed to get settlement by trade: %w", err)
	}
	return r.toDomain(&model), nil
}

func (r *settlementRepositoryImpl) toDomain(m *SettlementModel) *domain.Settlement {
	q, err := decimal.NewFromString(m.Quantity)
	if err != nil {
		q = decimal.Zero
	}
	p, err := decimal.NewFromString(m.Price)
	if err != nil {
		p = decimal.Zero
	}
	return &domain.Settlement{
		Model:          m.Model,
		SettlementID:   m.SettlementID,
		TradeID:        m.TradeID,
		BuyUserID:      m.BuyUserID,
		SellUserID:     m.SellUserID,
		Symbol:         m.Symbol,
		Quantity:       q,
		Price:          p,
		Status:         m.Status,
		SettlementTime: m.SettlementTime,
	}
}

// EODClearingModel 是日终清算的数据库模型。
type EODClearingModel struct {
	gorm.Model
	ClearingID    string     `gorm:"column:clearing_id;type:varchar(32);uniqueIndex;not null"`
	ClearingDate  string     `gorm:"column:clearing_date;type:varchar(20);index;not null"`
	Status        string     `gorm:"column:status;type:varchar(20);index;not null"`
	StartTime     time.Time  `gorm:"column:start_time;type:datetime;not null"`
	EndTime       *time.Time `gorm:"column:end_time;type:datetime"`
	TradesSettled int64      `gorm:"column:trades_settled;type:bigint;not null"`
	TotalTrades   int64      `gorm:"column:total_trades;type:bigint;not null"`
}

// TableName 指定表名
func (EODClearingModel) TableName() string {
	return "eod_clearings"
}

// eodClearingRepositoryImpl 是 domain.EODClearingRepository 接口的 GORM 实现。
type eodClearingRepositoryImpl struct {
	db *gorm.DB
}

// NewEODClearingRepository 创建日终清算仓储实例
func NewEODClearingRepository(db *gorm.DB) domain.EODClearingRepository {
	return &eodClearingRepositoryImpl{db: db}
}

// Save 实现 domain.EODClearingRepository.Save
func (r *eodClearingRepositoryImpl) Save(ctx context.Context, c *domain.EODClearing) error {
	model := r.fromEODDomain(c)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "clearing_id"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "eod_clearing_repository.Save failed", "clearing_id", c.ClearingID, "error", err)
		return fmt.Errorf("failed to save EOD clearing: %w", err)
	}
	c.Model = model.Model
	return nil
}

// Get 实现 domain.EODClearingRepository.Get
func (r *eodClearingRepositoryImpl) Get(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	var model EODClearingModel
	if err := r.db.WithContext(ctx).Where("clearing_id = ?", clearingID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "eod_clearing_repository.Get failed", "clearing_id", clearingID, "error", err)
		return nil, fmt.Errorf("failed to get EOD clearing: %w", err)
	}
	return r.toEODDomain(&model), nil
}

// GetLatest 实现 domain.EODClearingRepository.GetLatest
func (r *eodClearingRepositoryImpl) GetLatest(ctx context.Context) (*domain.EODClearing, error) {
	var model EODClearingModel
	if err := r.db.WithContext(ctx).Order("created_at desc").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "eod_clearing_repository.GetLatest failed", "error", err)
		return nil, fmt.Errorf("failed to get latest EOD clearing: %w", err)
	}
	return r.toEODDomain(&model), nil
}

// Update 实现 domain.EODClearingRepository.Update
func (r *eodClearingRepositoryImpl) Update(ctx context.Context, c *domain.EODClearing) error {
	model := r.fromEODDomain(c)
	err := r.db.WithContext(ctx).Model(&EODClearingModel{}).Where("clearing_id = ?", c.ClearingID).Updates(model).Error
	if err != nil {
		logging.Error(ctx, "eod_clearing_repository.Update failed", "clearing_id", c.ClearingID, "error", err)
		return fmt.Errorf("failed to update EOD clearing: %w", err)
	}
	return nil
}

func (r *eodClearingRepositoryImpl) fromEODDomain(c *domain.EODClearing) *EODClearingModel {
	return &EODClearingModel{
		Model:         c.Model,
		ClearingID:    c.ClearingID,
		ClearingDate:  c.ClearingDate,
		Status:        c.Status,
		StartTime:     c.StartTime,
		EndTime:       c.EndTime,
		TradesSettled: c.TradesSettled,
		TotalTrades:   c.TotalTrades,
	}
}

func (r *eodClearingRepositoryImpl) toEODDomain(m *EODClearingModel) *domain.EODClearing {
	return &domain.EODClearing{
		Model:         m.Model,
		ClearingID:    m.ClearingID,
		ClearingDate:  m.ClearingDate,
		Status:        m.Status,
		StartTime:     m.StartTime,
		EndTime:       m.EndTime,
		TradesSettled: m.TradesSettled,
		TotalTrades:   m.TotalTrades,
	}
}
