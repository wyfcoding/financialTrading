// 变更说明：重构 Custody 托管模型。
// 托管服务负责机构级资产的安全隔离、锁仓、红利派发以及份额变更记录。
package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type AssetStatus string

const (
	AssetAvailable AssetStatus = "AVAILABLE" // 可用
	AssetLocked     AssetStatus = "LOCKED"    // 锁仓（如IPO观察期、质押）
	AssetFrozen     AssetStatus = "FROZEN"    // 冻结（司法、合规检查）
	AssetInTransit  AssetStatus = "IN_TRANSIT" // 在途（清算期间）
)

type CustodyAsset struct {
	ID            uint64          `json:"id"`
	AccountID     string          `json:"account_id"`
	Symbol        string          `json:"symbol"`
	AssetClass    string          `json:"asset_class"` // STOCK, BOND, CASH, CRYPTO
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	AvailableQty  decimal.Decimal `json:"available_qty"`
	LockedQty     decimal.Decimal `json:"locked_qty"`
	FrozenQty     decimal.Decimal `json:"frozen_qty"`
	VaultID       string          `json:"vault_id"`    // 对应物理库房或链上地址
	Status        AssetStatus     `json:"status"`
	Version       int64           `json:"version"`
}

type AssetLock struct {
	ID          string          `json:"id"`
	AccountID   string          `json:"account_id"`
	Symbol      string          `json:"symbol"`
	Amount      decimal.Decimal `json:"amount"`
	LockType    string          `json:"lock_type"` // PLEDGE, IPO, JUDICIAL
	ReleaseTime *time.Time      `json:"release_time"`
	Metadata    map[string]string `json:"metadata"`
}

var (
	ErrInsufficientAsset = errors.New("insufficient asset quantity")
	ErrAssetNotFound     = errors.New("custody asset not found")
	ErrVaultUnavailable  = errors.New("designated vault is unavailable")
)
