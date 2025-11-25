// Package repository 包含仓储实现
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/clearing/domain"
	"github.com/wyfcoding/financialTrading/pkg/db"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"gorm.io/gorm"
)

// SettlementModel 清算记录数据库模型
type SettlementModel struct {
	gorm.Model
	// 清算 ID
	SettlementID string `gorm:"column:settlement_id;type:varchar(50);uniqueIndex;not null" json:"settlement_id"`
	// 交易 ID
	TradeID string `gorm:"column:trade_id;type:varchar(50);index;not null" json:"trade_id"`
	// 买方用户 ID
	BuyUserID string `gorm:"column:buy_user_id;type:varchar(50);index;not null" json:"buy_user_id"`
	// 卖方用户 ID
	SellUserID string `gorm:"column:sell_user_id;type:varchar(50);index;not null" json:"sell_user_id"`
	// 交易对
	Symbol string `gorm:"column:symbol;type:varchar(50);not null" json:"symbol"`
	// 成交数量
	Quantity string `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	// 成交价格
	Price string `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	// 清算状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// 清算时间
	SettlementTime int64 `gorm:"column:settlement_time;type:bigint;not null" json:"settlement_time"`
}

// TableName 指定表名
func (SettlementModel) TableName() string {
	return "settlements"
}

// SettlementRepositoryImpl 清算记录仓储实现
type SettlementRepositoryImpl struct {
	db *db.DB
}

// NewSettlementRepository 创建清算记录仓储
func NewSettlementRepository(database *db.DB) domain.SettlementRepository {
	return &SettlementRepositoryImpl{
		db: database,
	}
}

// Save 保存清算记录
func (sr *SettlementRepositoryImpl) Save(ctx context.Context, settlement *domain.Settlement) error {
	model := &SettlementModel{
		Model:          settlement.Model,
		SettlementID:   settlement.SettlementID,
		TradeID:        settlement.TradeID,
		BuyUserID:      settlement.BuyUserID,
		SellUserID:     settlement.SellUserID,
		Symbol:         settlement.Symbol,
		Quantity:       settlement.Quantity.String(),
		Price:          settlement.Price.String(),
		Status:         settlement.Status,
		SettlementTime: settlement.SettlementTime.Unix(),
	}

	if err := sr.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Error(ctx, "Failed to save settlement",
			"settlement_id", settlement.SettlementID,
			"error", err,
		)
		return fmt.Errorf("failed to save settlement: %w", err)
	}

	settlement.Model = model.Model
	return nil
}

// Get 获取清算记录
func (sr *SettlementRepositoryImpl) Get(ctx context.Context, settlementID string) (*domain.Settlement, error) {
	var model SettlementModel

	if err := sr.db.WithContext(ctx).Where("settlement_id = ?", settlementID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get settlement",
			"settlement_id", settlementID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	return sr.modelToDomain(&model), nil
}

// GetByUser 获取用户清算历史
func (sr *SettlementRepositoryImpl) GetByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Settlement, int64, error) {
	var models []SettlementModel
	var total int64

	query := sr.db.WithContext(ctx).Where("buy_user_id = ? OR sell_user_id = ?", userID, userID)

	if err := query.Model(&SettlementModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count settlements: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&models).Error; err != nil {
		logger.Error(ctx, "Failed to get settlements by user",
			"user_id", userID,
			"error", err,
		)
		return nil, 0, fmt.Errorf("failed to get settlements by user: %w", err)
	}

	settlements := make([]*domain.Settlement, 0, len(models))
	for _, model := range models {
		settlements = append(settlements, sr.modelToDomain(&model))
	}

	return settlements, total, nil
}

// GetByTrade 获取交易清算记录
func (sr *SettlementRepositoryImpl) GetByTrade(ctx context.Context, tradeID string) (*domain.Settlement, error) {
	var model SettlementModel

	if err := sr.db.WithContext(ctx).Where("trade_id = ?", tradeID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get settlement by trade",
			"trade_id", tradeID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get settlement by trade: %w", err)
	}

	return sr.modelToDomain(&model), nil
}

// modelToDomain 将数据库模型转换为领域对象
func (sr *SettlementRepositoryImpl) modelToDomain(model *SettlementModel) *domain.Settlement {
	quantity, _ := decimal.NewFromString(model.Quantity)
	price, _ := decimal.NewFromString(model.Price)

	return &domain.Settlement{
		Model:          model.Model,
		SettlementID:   model.SettlementID,
		TradeID:        model.TradeID,
		BuyUserID:      model.BuyUserID,
		SellUserID:     model.SellUserID,
		Symbol:         model.Symbol,
		Quantity:       quantity,
		Price:          price,
		Status:         model.Status,
		SettlementTime: time.Unix(model.SettlementTime, 0),
		CreatedAt:      model.CreatedAt,
	}
}

// EODClearingModel 日终清算数据库模型
type EODClearingModel struct {
	gorm.Model
	// 清算 ID
	ClearingID string `gorm:"column:clearing_id;type:varchar(50);uniqueIndex;not null" json:"clearing_id"`
	// 清算日期
	ClearingDate string `gorm:"column:clearing_date;type:varchar(20);index;not null" json:"clearing_date"`
	// 清算状态
	Status string `gorm:"column:status;type:varchar(20);index;not null" json:"status"`
	// 开始时间
	StartTime int64 `gorm:"column:start_time;type:bigint;not null" json:"start_time"`
	// 结束时间
	EndTime *int64 `gorm:"column:end_time;type:bigint" json:"end_time"`
	// 已清算交易数
	TradesSettled int64 `gorm:"column:trades_settled;type:bigint;not null" json:"trades_settled"`
	// 总交易数
	TotalTrades int64 `gorm:"column:total_trades;type:bigint;not null" json:"total_trades"`
}

// TableName 指定表名
func (EODClearingModel) TableName() string {
	return "eod_clearings"
}

// EODClearingRepositoryImpl 日终清算仓储实现
type EODClearingRepositoryImpl struct {
	db *db.DB
}

// NewEODClearingRepository 创建日终清算仓储
func NewEODClearingRepository(database *db.DB) domain.EODClearingRepository {
	return &EODClearingRepositoryImpl{
		db: database,
	}
}

// Save 保存日终清算
func (ecr *EODClearingRepositoryImpl) Save(ctx context.Context, clearing *domain.EODClearing) error {
	model := &EODClearingModel{
		Model:         clearing.Model,
		ClearingID:    clearing.ClearingID,
		ClearingDate:  clearing.ClearingDate,
		Status:        clearing.Status,
		StartTime:     clearing.StartTime.Unix(),
		TradesSettled: clearing.TradesSettled,
		TotalTrades:   clearing.TotalTrades,
	}

	if clearing.EndTime != nil {
		endTime := clearing.EndTime.Unix()
		model.EndTime = &endTime
	}

	if err := ecr.db.WithContext(ctx).Create(model).Error; err != nil {
		logger.Error(ctx, "Failed to save EOD clearing",
			"clearing_id", clearing.ClearingID,
			"error", err,
		)
		return fmt.Errorf("failed to save EOD clearing: %w", err)
	}

	clearing.Model = model.Model
	return nil
}

// Get 获取日终清算
func (ecr *EODClearingRepositoryImpl) Get(ctx context.Context, clearingID string) (*domain.EODClearing, error) {
	var model EODClearingModel

	if err := ecr.db.WithContext(ctx).Where("clearing_id = ?", clearingID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get EOD clearing",
			"clearing_id", clearingID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get EOD clearing: %w", err)
	}

	return ecr.modelToDomain(&model), nil
}

// GetLatest 获取最新日终清算
func (ecr *EODClearingRepositoryImpl) GetLatest(ctx context.Context) (*domain.EODClearing, error) {
	var model EODClearingModel

	if err := ecr.db.WithContext(ctx).Order("created_at DESC").First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error(ctx, "Failed to get latest EOD clearing", "error", err)
		return nil, fmt.Errorf("failed to get latest EOD clearing: %w", err)
	}

	return ecr.modelToDomain(&model), nil
}

// Update 更新日终清算
func (ecr *EODClearingRepositoryImpl) Update(ctx context.Context, clearing *domain.EODClearing) error {
	updates := map[string]interface{}{
		"status":         clearing.Status,
		"trades_settled": clearing.TradesSettled,
		"total_trades":   clearing.TotalTrades,
	}

	if clearing.EndTime != nil {
		updates["end_time"] = clearing.EndTime.Unix()
	}

	if err := ecr.db.WithContext(ctx).Model(&EODClearingModel{}).Where("clearing_id = ?", clearing.ClearingID).Updates(updates).Error; err != nil {
		logger.Error(ctx, "Failed to update EOD clearing",
			"clearing_id", clearing.ClearingID,
			"error", err,
		)
		return fmt.Errorf("failed to update EOD clearing: %w", err)
	}

	return nil
}

// modelToDomain 将数据库模型转换为领域对象
func (ecr *EODClearingRepositoryImpl) modelToDomain(model *EODClearingModel) *domain.EODClearing {
	var endTime *time.Time
	if model.EndTime != nil {
		t := time.Unix(*model.EndTime, 0)
		endTime = &t
	}

	return &domain.EODClearing{
		Model:         model.Model,
		ClearingID:    model.ClearingID,
		ClearingDate:  model.ClearingDate,
		Status:        model.Status,
		StartTime:     time.Unix(model.StartTime, 0),
		EndTime:       endTime,
		TradesSettled: model.TradesSettled,
		TotalTrades:   model.TotalTrades,
	}
}
