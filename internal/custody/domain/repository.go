package domain

import (
	"context"
)

type CustodyRepository interface {
	FindVaultByID(ctx context.Context, vaultID string) (*AssetVault, error)
	FindVaultByUser(ctx context.Context, userID uint64, symbol string) (*AssetVault, error)
	FindVaultByType(ctx context.Context, vaultType VaultType, symbol string) (*AssetVault, error)
	FindVaultsByUserID(ctx context.Context, userID uint64) ([]*AssetVault, error)
	ListVaultsByUser(ctx context.Context, userID uint64) ([]*AssetVault, error)
	ListVaultsByType(ctx context.Context, vaultType VaultType) ([]*AssetVault, error)
	ListVaultsBySymbol(ctx context.Context, symbol string) ([]*AssetVault, error)
	SaveVault(ctx context.Context, vault *AssetVault) error
	SaveTransfer(ctx context.Context, transfer *CustodyTransfer) error
	ListTransfersByVault(ctx context.Context, vaultID string, limit int) ([]*CustodyTransfer, error)
}

type CorpActionRepository interface {
	SaveAction(ctx context.Context, action *CorpAction) error
	FindActionByID(ctx context.Context, actionID string) (*CorpAction, error)
	ListActionsBySymbol(ctx context.Context, symbol string) ([]*CorpAction, error)
	ListPendingActions(ctx context.Context, symbol string) ([]*CorpAction, error)
	SaveExecution(ctx context.Context, execution *CorpActionExecution) error
}
