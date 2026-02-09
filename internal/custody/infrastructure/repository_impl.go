package infrastructure

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/custody/domain"
	"gorm.io/gorm"
)

type CustodyRepositoryImpl struct {
	db *gorm.DB
}

func NewCustodyRepository(db *gorm.DB) domain.CustodyRepository {
	return &CustodyRepositoryImpl{db: db}
}

func (r *CustodyRepositoryImpl) FindVaultByID(ctx context.Context, vaultID string) (*domain.AssetVault, error) {
	var vault domain.AssetVault
	if err := r.db.WithContext(ctx).Where("vault_id = ?", vaultID).First(&vault).Error; err != nil {
		return nil, err
	}
	return &vault, nil
}

func (r *CustodyRepositoryImpl) FindVaultsByUserID(ctx context.Context, userID uint64) ([]*domain.AssetVault, error) {
	var vaults []*domain.AssetVault
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&vaults).Error; err != nil {
		return nil, err
	}
	return vaults, nil
}

func (r *CustodyRepositoryImpl) SaveVault(ctx context.Context, vault *domain.AssetVault) error {
	return r.db.WithContext(ctx).Save(vault).Error
}

func (r *CustodyRepositoryImpl) SaveTransfer(ctx context.Context, transfer *domain.CustodyTransfer) error {
	return r.db.WithContext(ctx).Create(transfer).Error
}

type CorpActionRepositoryImpl struct {
	db *gorm.DB
}

func NewCorpActionRepository(db *gorm.DB) domain.CorpActionRepository {
	return &CorpActionRepositoryImpl{db: db}
}

func (r *CorpActionRepositoryImpl) SaveAction(ctx context.Context, action *domain.CorpAction) error {
	return r.db.WithContext(ctx).Save(action).Error
}

func (r *CorpActionRepositoryImpl) FindActionByID(ctx context.Context, actionID string) (*domain.CorpAction, error) {
	var action domain.CorpAction
	if err := r.db.WithContext(ctx).Where("action_id = ?", actionID).First(&action).Error; err != nil {
		return nil, err
	}
	return &action, nil
}

func (r *CorpActionRepositoryImpl) ListActionsBySymbol(ctx context.Context, symbol string) ([]*domain.CorpAction, error) {
	var actions []*domain.CorpAction
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Find(&actions).Error; err != nil {
		return nil, err
	}
	return actions, nil
}

func (r *CorpActionRepositoryImpl) SaveExecution(ctx context.Context, exec *domain.CorpActionExecution) error {
	return r.db.WithContext(ctx).Create(exec).Error
}
