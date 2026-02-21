// 变更说明：定义抵押品管理领域模型。
// 核心逻辑：应用 Haircut（折价率）将非现金资产转换为有效保证金。
package domain

import (
	"errors"
	"github.com/shopspring/decimal"
	"time"
)

type CollateralAsset struct {
	ID           uint64          `json:"id"`
	AccountID    string          `json:"account_id"`
	Symbol       string          `json:"symbol"`
	Quantity     decimal.Decimal `json:"quantity"`
	MarketPrice  decimal.Decimal `json:"market_price"`
	Haircut      decimal.Decimal `json:"haircut"` // 0.1 表示折价 10%，只算 90% 价值
	AssetType    string          `json:"asset_type"` // CASH, STOCK, BOND, CRYPTO
	UpdatedAt    time.Time       `json:"updated_at"`
}

// EffectiveValue 计算应用折价后的抵押品价值。
func (a *CollateralAsset) EffectiveValue() decimal.Decimal {
	marketValue := a.Quantity.Mul(a.MarketPrice)
	multiplier := decimal.NewFromInt(1).Sub(a.Haircut)
	return marketValue.Mul(multiplier)
}

type CollateralRepository interface {
	Save(asset *CollateralAsset) error
	GetAccountAssets(accountID string) ([]*CollateralAsset, error)
}

var (
	ErrInsufficientCollateral = errors.New("insufficient collateral")
)
