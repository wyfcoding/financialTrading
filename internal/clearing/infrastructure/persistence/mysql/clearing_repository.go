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

// SettlementModel 是清算记录的数据库模型，直接映射 settlements 表。
type SettlementModel struct {
	gorm.Model
	SettlementID   string    `gorm:"column:settlement_id;type:varchar(32);uniqueIndex;not null;comment:清算唯一ID"`
	TradeID        string    `gorm:"column:trade_id;type:varchar(32);index;not null;comment:关联成交ID"`
	BuyUserID      string    `gorm:"column:buy_user_id;type:varchar(32);index;not null;comment:买方用户ID"`
	SellUserID     string    `gorm:"column:sell_user_id;type:varchar(32);index;not null;comment:卖方用户ID"`
	Symbol         string    `gorm:"column:symbol;type:varchar(20);not null;comment:交易对"`
	Quantity       string    `gorm:"column:quantity;type:decimal(32,18);not null;comment:成交数量"`
	Price          string    `gorm:"column:price;type:decimal(32,18);not null;comment:成交价格"`
	Status         string    `gorm:"column:status;type:varchar(20);index;not null;comment:清算状态"`
	SettlementTime time.Time `gorm:"column:settlement_time;type:datetime;not null;comment:结算执行时间"`
}

// TableName 指定表名。
func (SettlementModel) TableName() string {
	return "settlements"
}

// settlementRepositoryImpl 是 domain.SettlementRepository 接口的 GORM 实现。
type settlementRepositoryImpl struct {
	db *gorm.DB
}

// NewSettlementRepository 创建并返回一个新的清算仓储实例。
func NewSettlementRepository(db *gorm.DB) domain.SettlementRepository {
	return &settlementRepositoryImpl{db: db}
}

// Save 持久化结算记录，支持冲突时的全量同步。
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
		logging.Error(ctx, "settlement_repository.save failed", "settlement_id", s.SettlementID, "error", err)
		return fmt.Errorf("failed to save settlement: %w", err)
	}
	s.Model = model.Model
	return nil
}

// Get 根据结算 ID 获取详情。
func (r *settlementRepositoryImpl) Get(ctx context.Context, settlementID string) (*domain.Settlement, error) {
	var model SettlementModel
	if err := r.db.WithContext(ctx).Where("settlement_id = ?", settlementID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "settlement_repository.get failed", "settlement_id", settlementID, "error", err)
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}
	return r.toDomain(&model), nil
}

// GetByUser 分页获取指定用户的历史结算记录。
func (r *settlementRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Settlement, int64, error) {
	var models []SettlementModel
	var total int64
	db := r.db.WithContext(ctx).Model(&SettlementModel{}).Where("buy_user_id = ? OR sell_user_id = ?", userID, userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		logging.Error(ctx, "settlement_repository.get_by_user failed", "user_id", userID, "error", err)
		return nil, 0, fmt.Errorf("failed to get settlements by user: %w", err)
	}
	res := make([]*domain.Settlement, len(models))
	for i, m := range models {
		res[i] = r.toDomain(&m)
	}
	return res, total, nil
}

// GetByTrade 根据成交 ID 精准查询结算记录。
func (r *settlementRepositoryImpl) GetByTrade(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var model SettlementModel
	if err := r.db.WithContext(ctx).Where("trade_id = ?", tradeID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "settlement_repository.get_by_trade failed", "trade_id", tradeID, "error", err)
		return nil, fmt.Errorf("failed to get settlement by trade: %w", err)
	}
	return r.toDomain(&model), nil
}

func (r *settlementRepositoryImpl) toDomain(m *SettlementModel) *domain.Settlement {
	q, _ := decimal.NewFromString(m.Quantity)
	p, _ := decimal.NewFromString(m.Price)
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

// EODClearingModel 是日终清算的数据库模型，记录批次状态。
type EODClearingModel struct {
	gorm.Model
	ClearingID    string     `gorm:"column:clearing_id;type:varchar(32);uniqueIndex;not null;comment:批次ID"`
	ClearingDate  string     `gorm:"column:clearing_date;type:varchar(20);index;not null;comment:清算日期"`
	Status        string     `gorm:"column:status;type:varchar(20);index;not null;comment:批次状态"`
	StartTime     time.Time  `gorm:"column:start_time;type:datetime;not null;comment:开始时间"`
	EndTime       *time.Time `gorm:"column:end_time;type:datetime;comment:结束时间"`
	TradesSettled int64      `gorm:"column:trades_settled;type:bigint;not null;comment:已结算数"`
	TotalTrades   int64      `gorm:"column:total_trades;type:bigint;not null;comment:总成交数"`
}

// TableName 指定表名。
func (EODClearingModel) TableName() string {
	return "eod_clearings"
}

// eodClearingRepositoryImpl 实现了日终清算仓储接口。
type eodClearingRepositoryImpl struct {
	db *gorm.DB
}

// NewEODClearingRepository 构造一个新的日终清算仓储实例。
func NewEODClearingRepository(db *gorm.DB) domain.EODClearingRepository {
	return &eodClearingRepositoryImpl{db: db}
}

// Save 持久化日终清算状态。
func (r *eodClearingRepositoryImpl) Save(ctx context.Context, c *domain.EODClearing) error {
	model := r.fromEODDomain(c)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "clearing_id"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "eod_clearing_repository.save failed", "clearing_id", c.ClearingID, "error", err)
		return fmt.Errorf("failed to save eod clearing: %w", err)
	}
	c.Model = model.Model
	return nil
}

// Get 获取指定的清算批次详情。
func (r *eodClearingRepositoryImpl) Get(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	var model EODClearingModel
	if err := r.db.WithContext(ctx).Where("clearing_id = ?", clearingID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "eod_clearing_repository.get failed", "clearing_id", clearingID, "error", err)
		return nil, fmt.Errorf("failed to get eod clearing: %w", err)
	}
	return r.toEODDomain(&model), nil
}

// GetLatest 获取最近一次执行的清算批次。
func (r *eodClearingRepositoryImpl) GetLatest(ctx context.Context) (*domain.EODClearing, error) {
	var model EODClearingModel
	if err := r.db.WithContext(ctx).Order("created_at desc").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "eod_clearing_repository.get_latest failed", "error", err)
		return nil, fmt.Errorf("failed to get latest eod clearing: %w", err)
	}
	return r.toEODDomain(&model), nil
}

// Update 更新清算批次的执行进度或状态。
func (r *eodClearingRepositoryImpl) Update(ctx context.Context, c *domain.EODClearing) error {
	model := r.fromEODDomain(c)
	err := r.db.WithContext(ctx).Model(&EODClearingModel{}).Where("clearing_id = ?", c.ClearingID).Updates(model).Error
	if err != nil {
		logging.Error(ctx, "eod_clearing_repository.update failed", "clearing_id", c.ClearingID, "error", err)
		return fmt.Errorf("failed to update eod clearing: %w", err)
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

// MarginRequirementModel 是保证金要求的数据库模型。
type MarginRequirementModel struct {
	gorm.Model
	Symbol           string `gorm:"column:symbol;type:varchar(20);uniqueIndex;not null;comment:交易对"`
	BaseMarginRate   string `gorm:"column:base_margin_rate;type:decimal(10,4);not null;comment:基础率"`
	VolatilityFactor string `gorm:"column:volatility_factor;type:decimal(10,4);not null;comment:波动因子"`
	UpdatedBy        string `gorm:"column:updated_by;type:varchar(32);comment:更新人"`
}

// TableName 指定表名。
func (MarginRequirementModel) TableName() string {
	return "margin_requirements"
}

// marginRequirementRepositoryImpl 实现了保证金规则仓储接口。
type marginRequirementRepositoryImpl struct {
	db *gorm.DB
}

// NewMarginRequirementRepository 构造一个新的保证金规则仓储实例。
func NewMarginRequirementRepository(db *gorm.DB) domain.MarginRequirementRepository {
	return &marginRequirementRepositoryImpl{db: db}
}

// Save 持久化保证金配置。
func (r *marginRequirementRepositoryImpl) Save(ctx context.Context, m *domain.MarginRequirement) error {
	model := &MarginRequirementModel{
		Model:            m.Model,
		Symbol:           m.Symbol,
		BaseMarginRate:   m.BaseMarginRate.String(),
		VolatilityFactor: m.VolatilityFactor.String(),
		UpdatedBy:        m.UpdatedBy,
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}},
		UpdateAll: true,
	}).Create(model).Error
	if err != nil {
		logging.Error(ctx, "margin_requirement_repository.save failed", "symbol", m.Symbol, "error", err)
		return fmt.Errorf("failed to save margin requirement: %w", err)
	}
	m.Model = model.Model
	return nil
}

// GetBySymbol 根据交易对检索配置。
func (r *marginRequirementRepositoryImpl) GetBySymbol(ctx context.Context, symbol string) (*domain.MarginRequirement, error) {
	var model MarginRequirementModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logging.Error(ctx, "margin_requirement_repository.get_by_symbol failed", "symbol", symbol, "error", err)
		return nil, fmt.Errorf("failed to get margin requirement: %w", err)
	}

	baseRate, err := decimal.NewFromString(model.BaseMarginRate)
	if err != nil {
		return nil, fmt.Errorf("invalid base_rate in db: %w", err)
	}
	volFactor, err := decimal.NewFromString(model.VolatilityFactor)
	if err != nil {
		return nil, fmt.Errorf("invalid volatility_factor in db: %w", err)
	}

	return &domain.MarginRequirement{
		Model:            model.Model,
		Symbol:           model.Symbol,
		BaseMarginRate:   baseRate,
		VolatilityFactor: volFactor,
		UpdatedBy:        model.UpdatedBy,
	}, nil
}
