package domain

import (
	"context"
	"fmt"
	"time"
)

type VaultType string

const (
	VaultCustomer VaultType = "CUSTOMER"
	VaultHouse    VaultType = "HOUSE"
	VaultOmnibus  VaultType = "OMNIBUS"
)

type AssetVault struct {
	VaultID   string    `json:"vault_id"`
	Type      VaultType `json:"type"`
	UserID    uint64    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Balance   int64     `json:"balance"`
	Locked    int64     `json:"locked"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewCustomerVault(userID uint64, symbol string) *AssetVault {
	return &AssetVault{
		VaultID:   fmt.Sprintf("CUST-%d-%s", userID, symbol),
		Type:      VaultCustomer,
		UserID:    userID,
		Symbol:    symbol,
		Balance:   0,
		Locked:    0,
		UpdatedAt: time.Now(),
	}
}

func NewHouseVault(symbol string) *AssetVault {
	return &AssetVault{
		VaultID:   fmt.Sprintf("HOUSE-%s", symbol),
		Type:      VaultHouse,
		Symbol:    symbol,
		UpdatedAt: time.Now(),
	}
}

func NewOmnibusVault(symbol string) *AssetVault {
	return &AssetVault{
		VaultID:   fmt.Sprintf("OMNI-%s", symbol),
		Type:      VaultOmnibus,
		Symbol:    symbol,
		UpdatedAt: time.Now(),
	}
}

func (v *AssetVault) AvailableBalance() int64 {
	return v.Balance - v.Locked
}

func (v *AssetVault) SafeDebit(amount int64) error {
	if v.AvailableBalance() < amount {
		return fmt.Errorf("insufficient available balance in vault %s: available=%d, requested=%d",
			v.VaultID, v.AvailableBalance(), amount)
	}
	v.Balance -= amount
	v.UpdatedAt = time.Now()
	return nil
}

func (v *AssetVault) SafeCredit(amount int64) {
	v.Balance += amount
	v.UpdatedAt = time.Now()
}

func (v *AssetVault) Lock(amount int64) error {
	if v.AvailableBalance() < amount {
		return fmt.Errorf("insufficient available balance to lock")
	}
	v.Locked += amount
	v.UpdatedAt = time.Now()
	return nil
}

func (v *AssetVault) Unlock(amount int64) {
	if v.Locked < amount {
		amount = v.Locked
	}
	v.Locked -= amount
	v.UpdatedAt = time.Now()
}

func (v *AssetVault) IsCustomerVault() bool {
	return v.Type == VaultCustomer
}

func (v *AssetVault) IsHouseVault() bool {
	return v.Type == VaultHouse
}

func (v *AssetVault) IsOmnibusVault() bool {
	return v.Type == VaultOmnibus
}

type CustodyTransfer struct {
	TransferID string    `json:"transfer_id"`
	FromVault  string    `json:"from_vault"`
	ToVault    string    `json:"to_vault"`
	Symbol     string    `json:"symbol"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason"`
	Timestamp  time.Time `json:"timestamp"`
}

func NewCustodyTransfer(from, to, symbol string, amount int64, reason string) *CustodyTransfer {
	return &CustodyTransfer{
		TransferID: fmt.Sprintf("TX-%d", time.Now().UnixNano()),
		FromVault:  from,
		ToVault:    to,
		Symbol:     symbol,
		Amount:     amount,
		Reason:     reason,
		Timestamp:  time.Now(),
	}
}

type CustodyService interface {
	TransferInternal(ctx context.Context, from, to string, amount int64, reason string) error
	Segregate(ctx context.Context, userID uint64) error
	GetHolding(ctx context.Context, vaultID string) (*AssetVault, error)
}
