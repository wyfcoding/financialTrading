// Package mysql 抵押品服务 MySQL 仓储实现
package mysql

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/collateral/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/database"
	"gorm.io/gorm"
)

type CollateralRepositoryImpl struct {
	db *database.DB
}

func NewCollateralRepository(db *database.DB) domain.CollateralRepository {
	return &CollateralRepositoryImpl{db: db}
}

func (r *CollateralRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return r.db.DB.WithContext(ctx)
}

func (r *CollateralRepositoryImpl) Save(ctx context.Context, asset *domain.CollateralAsset) error {
	return r.getDB(ctx).Save(asset).Error
}

func (r *CollateralRepositoryImpl) GetByAssetID(ctx context.Context, assetID string) (*domain.CollateralAsset, error) {
	var asset domain.CollateralAsset
	err := r.getDB(ctx).Where("asset_id = ?", assetID).First(&asset).Error
	return &asset, err
}

func (r *CollateralRepositoryImpl) GetByAccountAndSymbol(ctx context.Context, accountID, symbol string) (*domain.CollateralAsset, error) {
	var asset domain.CollateralAsset
	err := r.getDB(ctx).Where("account_id = ? AND symbol = ? AND status = 'ACTIVE'", accountID, symbol).First(&asset).Error
	return &asset, err
}

func (r *CollateralRepositoryImpl) ListByAccount(ctx context.Context, accountID string) ([]*domain.CollateralAsset, error) {
	var assets []*domain.CollateralAsset
	err := r.getDB(ctx).Where("account_id = ? AND status = 'ACTIVE'", accountID).Find(&assets).Error
	return assets, err
}

func (r *CollateralRepositoryImpl) GetTotalCollateralValue(ctx context.Context, accountID, currency string) (decimal.Decimal, error) {
	var total decimal.Decimal
	// 简单的 SUM 查询，实际可能需要处理汇率转换，这里假设同一币种或已在业务层过滤
	err := r.getDB(ctx).Model(&domain.CollateralAsset{}).
		Where("account_id = ? AND currency = ? AND status = 'ACTIVE'", accountID, currency).
		Select("COALESCE(SUM(collateral_value), 0)").
		Scan(&total).Error
	return total, err
}

type HaircutRepositoryImpl struct {
	db *database.DB
}

func NewHaircutRepository(db *database.DB) domain.HaircutRepository {
	return &HaircutRepositoryImpl{db: db}
}

func (r *HaircutRepositoryImpl) GetSchedule(ctx context.Context, assetType domain.AssetType, symbol string) (*domain.HaircutSchedule, error) {
	var schedule domain.HaircutSchedule
	// 优先匹配 Symbol，其次匹配 AssetType
	err := r.db.DB.WithContext(ctx).
		Where("(symbol = ? OR (symbol = '' AND asset_type = ?)) AND is_eligible = true", symbol, assetType).
		Order("symbol DESC"). // 非空 symbol 优先
		First(&schedule).Error
	return &schedule, err
}

func (r *HaircutRepositoryImpl) Save(ctx context.Context, schedule *domain.HaircutSchedule) error {
	return r.db.DB.WithContext(ctx).Save(schedule).Error
}

type AllocationRepositoryImpl struct {
	db *database.DB
}

func NewAllocationRepository(db *database.DB) domain.AllocationRepository {
	return &AllocationRepositoryImpl{db: db}
}

func (r *AllocationRepositoryImpl) Save(ctx context.Context, alloc *domain.Allocation) error {
	return r.db.DB.WithContext(ctx).Save(alloc).Error
}

func (r *AllocationRepositoryImpl) ListByAssetID(ctx context.Context, assetID string) ([]*domain.Allocation, error) {
	var list []*domain.Allocation
	err := r.db.DB.WithContext(ctx).Where("asset_id = ? AND status = 'ALLOCATED'", assetID).Find(&list).Error
	return list, err
}
