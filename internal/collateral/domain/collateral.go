// Package domain 抵押品管理领域模型
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrCollateralNotFound = errors.New("collateral asset not found")
	ErrInsufficientCollateral = errors.New("insufficient collateral")
	ErrAssetNotEligible = errors.New("asset not eligible for collateral")
)

type AssetType string

const (
	AssetTypeCash       AssetType = "CASH"
	AssetTypeGovernment AssetType = "GOVERNMENT_BOND"
	AssetTypeCorporate  AssetType = "CORPORATE_BOND"
	AssetTypeEquity     AssetType = "EQUITY"
	AssetTypeLetter     AssetType = "LETTER_OF_CREDIT" // 信用证
)

// CollateralAsset 抵押品资产聚合根
// 代表某个账户持有的、用于通过保证金测试的资产
type CollateralAsset struct {
	gorm.Model
	AssetID       string          `gorm:"column:asset_id;type:varchar(64);uniqueIndex;not null"`
	AccountID     string          `gorm:"column:account_id;index;not null"` // 保证金账户ID
	AssetType     AssetType       `gorm:"column:asset_type;type:varchar(32);not null"`
	Symbol        string          `gorm:"column:symbol;type:varchar(32);not null"` // 现金则为货币代码，证券则为Ticker
	Quantity      decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null"`
	MarketPrice   decimal.Decimal `gorm:"column:market_price;type:decimal(20,8);not null"`
	MarketValue   decimal.Decimal `gorm:"column:market_value;type:decimal(20,2);not null"` // 市场价值
	Haircut       decimal.Decimal `gorm:"column:haircut;type:decimal(5,4);not null"`       // 折扣率 (0.05 = 5%)
	CollateralValue decimal.Decimal `gorm:"column:collateral_value;type:decimal(20,2);not null"` // 折后价值
	Currency      string          `gorm:"column:currency;type:char(3);not null"`
	Status        string          `gorm:"column:status;type:varchar(32);default:'ACTIVE'"` // ACTIVE, RELEASED, LIQUIDATED
	LastValuation time.Time       `gorm:"column:last_valuation"`
}

func (CollateralAsset) TableName() string { return "collateral_assets" }

// HaircutSchedule 折扣率规则表
type HaircutSchedule struct {
	gorm.Model
	Symbol        string          `gorm:"column:symbol;type:varchar(32);index"` // 特定品种
	AssetType     AssetType       `gorm:"column:asset_type;type:varchar(32);index"`
	MinRating     string          `gorm:"column:min_rating;type:varchar(16)"`   // 最低评级要求
	BaseHaircut   decimal.Decimal `gorm:"column:base_haircut;type:decimal(5,4);not null"`
	VolatilityAdj decimal.Decimal `gorm:"column:volatility_adj;type:decimal(5,4)"` // 波动率调整
	Currency      string          `gorm:"column:currency;type:char(3)"`
	IsEligible    bool            `gorm:"column:is_eligible;default:true"`
}

func (HaircutSchedule) TableName() string { return "collateral_haircut_schedules" }

// Allocation 抵押品分配记录 (被哪个义务占用)
type Allocation struct {
	gorm.Model
	AllocationID  string          `gorm:"column:allocation_id;type:varchar(64);uniqueIndex"`
	AssetID       string          `gorm:"column:asset_id;index;not null"`
	ObligationID  string          `gorm:"column:obligation_id;index;not null"` // 如 MarginCallID 或 TradeID
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(20,2);not null"`
	Status        string          `gorm:"column:status;type:varchar(32);default:'ALLOCATED'"`
}

func (Allocation) TableName() string { return "collateral_allocations" }

// NewCollateralAsset 创建新抵押品
func NewCollateralAsset(accountID string, assetType AssetType, symbol string, qty decimal.Decimal, currency string) *CollateralAsset {
	return &CollateralAsset{
		AccountID: accountID,
		AssetType: assetType,
		Symbol:    symbol,
		Quantity:  qty,
		Currency:  currency,
		Status:    "ACTIVE",
	}
}

// UpdateValuation 更新估值
func (c *CollateralAsset) UpdateValuation(price decimal.Decimal, haircut decimal.Decimal) {
	c.MarketPrice = price
	c.MarketValue = c.Quantity.Mul(price)
	c.Haircut = haircut
	// Collateral Value = Market Value * (1 - Haircut)
	c.CollateralValue = c.MarketValue.Mul(decimal.NewFromInt(1).Sub(haircut))
	c.LastValuation = time.Now()
}

// Deposit 增加数量
func (c *CollateralAsset) Deposit(qty decimal.Decimal) {
	c.Quantity = c.Quantity.Add(qty)
}

// Withdraw 减少数量
func (c *CollateralAsset) Withdraw(qty decimal.Decimal) error {
	if c.Quantity.LessThan(qty) {
		return ErrInsufficientCollateral
	}
	c.Quantity = c.Quantity.Sub(qty)
	
	// 如果数量为0，可能会标记为 RELEASED，这里简化处理保留 ACTIVE 但 qty=0
	return nil
}

// Repositories
type CollateralRepository interface {
	Save(ctx context.Context, asset *CollateralAsset) error
	GetByAssetID(ctx context.Context, assetID string) (*CollateralAsset, error)
	GetByAccountAndSymbol(ctx context.Context, accountID, symbol string) (*CollateralAsset, error)
	ListByAccount(ctx context.Context, accountID string) ([]*CollateralAsset, error)
	GetTotalCollateralValue(ctx context.Context, accountID, currency string) (decimal.Decimal, error)
}

type HaircutRepository interface {
	GetSchedule(ctx context.Context, assetType AssetType, symbol string) (*HaircutSchedule, error)
	Save(ctx context.Context, schedule *HaircutSchedule) error
}

type AllocationRepository interface {
	Save(ctx context.Context, alloc *Allocation) error
	ListByAssetID(ctx context.Context, assetID string) ([]*Allocation, error)
}
