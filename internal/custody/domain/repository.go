package domain

import (
	"context"
)

// CustodyRepository 托管仓储接口
type CustodyRepository interface {
	FindVaultByID(ctx context.Context, vaultID string) (*AssetVault, error)
	FindVaultsByUserID(ctx context.Context, userID uint64) ([]*AssetVault, error)
	SaveVault(ctx context.Context, vault *AssetVault) error
	SaveTransfer(ctx context.Context, transfer *CustodyTransfer) error
}

// CorpActionRepository 公司行为仓储接口
type CorpActionRepository interface {
	SaveAction(ctx context.Context, action *CorpAction) error
	FindActionByID(ctx context.Context, actionID string) (*CorpAction, error)
	ListActionsBySymbol(ctx context.Context, symbol string) ([]*CorpAction, error)
	SaveExecution(ctx context.Context, execution *CorpActionExecution) error
}
