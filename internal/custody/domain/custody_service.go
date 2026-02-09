// Package domain 提供了资产托管（Custody）领域的业务逻辑。
// 变更说明：实现客户资产库（Custody Vault）与法律隔离账本逻辑，确保客户资产与公司自有资金的严格区分。
package domain

import (
	"context"
	"fmt"
	"time"
)

// VaultType 库位类型
type VaultType string

const (
	VaultCustomer VaultType = "CUSTOMER" // 客户隔离库
	VaultHouse    VaultType = "HOUSE"    // 公司自有库
	VaultOmnibus  VaultType = "OMNIBUS"  // 综合账户（用于对接外部清算）
)

// AssetVault 资产库实体
type AssetVault struct {
	VaultID   string    `json:"vault_id"`
	Type      VaultType `json:"type"`
	UserID    uint64    `json:"user_id"` // 如果是 CUSTOMER 类型
	Symbol    string    `json:"symbol"`
	Balance   int64     `json:"balance"`
	Locked    int64     `json:"locked"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CustodyTransfer 托管转移流水
type CustodyTransfer struct {
	TransferID string    `json:"transfer_id"`
	FromVault  string    `json:"from_vault"`
	ToVault    string    `json:"to_vault"`
	Symbol     string    `json:"symbol"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason"` // 如：Trade Settlement, Deposit, Withdrawal
	Timestamp  time.Time `json:"timestamp"`
}

// CustodyService 托管服务接口
type CustodyService interface {
	// TransferInternal 内部库位间转移（如交易结算时的资产搬运）
	TransferInternal(ctx context.Context, from, to string, amount int64, reason string) error
	// Segregate 确保客户资产按法规隔离
	Segregate(ctx context.Context, userID uint64) error
	// GetHolding 查看托管持仓
	GetHolding(ctx context.Context, vaultID string) (*AssetVault, error)
}

// VaultManager 库位管理逻辑
type VaultManager struct{}

// SafeDebit 安全扣减（需校验隔离原则）
func (v *AssetVault) SafeDebit(amount int64) error {
	if v.Balance-v.Locked < amount {
		return fmt.Errorf("insufficient available balance in vault %s", v.VaultID)
	}
	v.Balance -= amount
	v.UpdatedAt = time.Now()
	return nil
}

// SafeCredit 安全入账
func (v *AssetVault) SafeCredit(amount int64) {
	v.Balance += amount
	v.UpdatedAt = time.Now()
}
