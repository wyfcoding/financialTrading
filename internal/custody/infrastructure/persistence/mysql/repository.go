package mysql

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/custody/domain"
	"gorm.io/gorm"
)

type CustodyRepo struct {
	db *gorm.DB
}

func NewCustodyRepo(db *gorm.DB) domain.CustodyRepository {
	return &CustodyRepo{db: db}
}

func (r *CustodyRepo) FindVaultByID(ctx context.Context, vaultID string) (*domain.AssetVault, error) {
	var model AssetVaultModel
	if err := r.db.WithContext(ctx).Where("vault_id = ?", vaultID).First(&model).Error; err != nil {
		return nil, err
	}
	return toDomainVault(&model), nil
}

func (r *CustodyRepo) FindVaultsByUserID(ctx context.Context, userID uint64) ([]*domain.AssetVault, error) {
	var models []AssetVaultModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	var vaults []*domain.AssetVault
	for _, m := range models {
		vaults = append(vaults, toDomainVault(&m))
	}
	return vaults, nil
}

func (r *CustodyRepo) SaveVault(ctx context.Context, vault *domain.AssetVault) error {
	model := AssetVaultModel{
		VaultID: vault.VaultID,
		Type:    string(vault.Type),
		UserID:  vault.UserID,
		Symbol:  vault.Symbol,
		Balance: vault.Balance,
		Locked:  vault.Locked,
	}
	// Check exist for ID
	var exist AssetVaultModel
	if err := r.db.WithContext(ctx).Where("vault_id = ?", vault.VaultID).First(&exist).Error; err == nil {
		model.ID = exist.ID
		model.CreatedAt = exist.CreatedAt
	}
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *CustodyRepo) SaveTransfer(ctx context.Context, transfer *domain.CustodyTransfer) error {
	model := CustodyTransferModel{
		TransferID: transfer.TransferID,
		FromVault:  transfer.FromVault,
		ToVault:    transfer.ToVault,
		Symbol:     transfer.Symbol,
		Amount:     transfer.Amount,
		Reason:     transfer.Reason,
		Timestamp:  transfer.Timestamp,
	}
	return r.db.WithContext(ctx).Save(&model).Error
}

// Helper
func toDomainVault(m *AssetVaultModel) *domain.AssetVault {
	return &domain.AssetVault{
		VaultID:   m.VaultID,
		Type:      domain.VaultType(m.Type),
		UserID:    m.UserID,
		Symbol:    m.Symbol,
		Balance:   m.Balance,
		Locked:    m.Locked,
		UpdatedAt: m.UpdatedAt,
	}
}

// CorpActionRepo impl
type CorpActionRepo struct {
	db *gorm.DB
}

func NewCorpActionRepo(db *gorm.DB) domain.CorpActionRepository {
	return &CorpActionRepo{db: db}
}

func (r *CorpActionRepo) SaveAction(ctx context.Context, action *domain.CorpAction) error {
	model := CorpActionModel{
		ActionID:   action.ActionID,
		Symbol:     action.Symbol,
		Type:       string(action.Type),
		Ratio:      action.Ratio,
		RecordDate: action.RecordDate,
		ExDate:     action.ExDate,
		PayDate:    action.PayDate,
		Status:     action.Status,
	}
	var exist CorpActionModel
	if err := r.db.WithContext(ctx).Where("action_id = ?", action.ActionID).First(&exist).Error; err == nil {
		model.ID = exist.ID
		model.CreatedAt = exist.CreatedAt
	}
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *CorpActionRepo) FindActionByID(ctx context.Context, actionID string) (*domain.CorpAction, error) {
	var model CorpActionModel
	if err := r.db.WithContext(ctx).Where("action_id = ?", actionID).First(&model).Error; err != nil {
		return nil, err
	}
	return toDomainAction(&model), nil
}

func (r *CorpActionRepo) ListActionsBySymbol(ctx context.Context, symbol string) ([]*domain.CorpAction, error) {
	var models []CorpActionModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Find(&models).Error; err != nil {
		return nil, err
	}
	var actions []*domain.CorpAction
	for _, m := range models {
		actions = append(actions, toDomainAction(&m))
	}
	return actions, nil
}

func (r *CorpActionRepo) SaveExecution(ctx context.Context, execution *domain.CorpActionExecution) error {
	model := CorpActionExecutionModel{
		ExecutionID: execution.ExecutionID,
		ActionID:    execution.ActionID,
		UserID:      execution.UserID,
		OldPosition: execution.OldPosition,
		NewPosition: execution.NewPosition,
		ChangeAmt:   execution.ChangeAmt,
		ExecutedAt:  execution.ExecutedAt,
	}
	return r.db.WithContext(ctx).Save(&model).Error
}

func toDomainAction(m *CorpActionModel) *domain.CorpAction {
	return &domain.CorpAction{
		ActionID:   m.ActionID,
		Symbol:     m.Symbol,
		Type:       domain.CorpActionType(m.Type),
		Ratio:      m.Ratio,
		RecordDate: m.RecordDate,
		ExDate:     m.ExDate,
		PayDate:    m.PayDate,
		Status:     m.Status,
	}
}
