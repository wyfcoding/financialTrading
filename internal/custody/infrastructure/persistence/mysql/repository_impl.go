package mysql

import (
	"context"
	"errors"

	"github.com/wyfcoding/financialtrading/internal/custody/domain"
	"gorm.io/gorm"
)

type custodyRepo struct {
	db *gorm.DB
}

type corpActionRepo struct {
	db *gorm.DB
}

func NewCustodyRepo(db *gorm.DB) domain.CustodyRepository {
	return &custodyRepo{db: db}
}

func NewCorpActionRepo(db *gorm.DB) domain.CorpActionRepository {
	return &corpActionRepo{db: db}
}

func (r *custodyRepo) FindVaultByID(ctx context.Context, vaultID string) (*domain.AssetVault, error) {
	var model AssetVaultModel
	if err := r.db.WithContext(ctx).Where("vault_id = ?", vaultID).First(&model).Error; err != nil {
		return nil, err
	}
	return assetVaultToDomain(&model), nil
}

func (r *custodyRepo) FindVaultByUser(ctx context.Context, userID uint64, symbol string) (*domain.AssetVault, error) {
	var model AssetVaultModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol = ? AND type = ?", userID, symbol, string(domain.VaultCustomer)).
		First(&model).Error; err != nil {
		return nil, err
	}
	return assetVaultToDomain(&model), nil
}

func (r *custodyRepo) FindVaultByType(ctx context.Context, vaultType domain.VaultType, symbol string) (*domain.AssetVault, error) {
	var model AssetVaultModel
	if err := r.db.WithContext(ctx).
		Where("type = ? AND symbol = ?", string(vaultType), symbol).
		First(&model).Error; err != nil {
		return nil, err
	}
	return assetVaultToDomain(&model), nil
}

func (r *custodyRepo) FindVaultsByUserID(ctx context.Context, userID uint64) ([]*domain.AssetVault, error) {
	return r.ListVaultsByUser(ctx, userID)
}

func (r *custodyRepo) ListVaultsByUser(ctx context.Context, userID uint64) ([]*domain.AssetVault, error) {
	var models []AssetVaultModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	return mapAssetVaults(models), nil
}

func (r *custodyRepo) ListVaultsByType(ctx context.Context, vaultType domain.VaultType) ([]*domain.AssetVault, error) {
	var models []AssetVaultModel
	if err := r.db.WithContext(ctx).Where("type = ?", string(vaultType)).Find(&models).Error; err != nil {
		return nil, err
	}
	return mapAssetVaults(models), nil
}

func (r *custodyRepo) ListVaultsBySymbol(ctx context.Context, symbol string) ([]*domain.AssetVault, error) {
	var models []AssetVaultModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Find(&models).Error; err != nil {
		return nil, err
	}
	return mapAssetVaults(models), nil
}

func (r *custodyRepo) SaveVault(ctx context.Context, vault *domain.AssetVault) error {
	var existing AssetVaultModel
	err := r.db.WithContext(ctx).Where("vault_id = ?", vault.VaultID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(domainVaultToModel(vault)).Error
		}
		return err
	}

	existing.Type = string(vault.Type)
	existing.UserID = vault.UserID
	existing.Symbol = vault.Symbol
	existing.Balance = vault.Balance
	existing.Locked = vault.Locked
	return r.db.WithContext(ctx).Save(&existing).Error
}

func (r *custodyRepo) SaveTransfer(ctx context.Context, transfer *domain.CustodyTransfer) error {
	return r.db.WithContext(ctx).Create(domainTransferToModel(transfer)).Error
}

func (r *custodyRepo) ListTransfersByVault(ctx context.Context, vaultID string, limit int) ([]*domain.CustodyTransfer, error) {
	query := r.db.WithContext(ctx).
		Where("from_vault = ? OR to_vault = ?", vaultID, vaultID).
		Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var models []CustodyTransferModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	transfers := make([]*domain.CustodyTransfer, 0, len(models))
	for i := range models {
		transfers = append(transfers, transferToDomain(&models[i]))
	}
	return transfers, nil
}

func (r *corpActionRepo) SaveAction(ctx context.Context, action *domain.CorpAction) error {
	var existing CorpActionModel
	err := r.db.WithContext(ctx).Where("action_id = ?", action.ActionID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(domainActionToModel(action)).Error
		}
		return err
	}

	existing.Symbol = action.Symbol
	existing.Type = string(action.Type)
	existing.Ratio = action.Ratio
	existing.RecordDate = action.RecordDate
	existing.ExDate = action.ExDate
	existing.PayDate = action.PayDate
	existing.Status = action.Status
	return r.db.WithContext(ctx).Save(&existing).Error
}

func (r *corpActionRepo) FindActionByID(ctx context.Context, actionID string) (*domain.CorpAction, error) {
	var model CorpActionModel
	if err := r.db.WithContext(ctx).Where("action_id = ?", actionID).First(&model).Error; err != nil {
		return nil, err
	}
	return actionToDomain(&model), nil
}

func (r *corpActionRepo) ListActionsBySymbol(ctx context.Context, symbol string) ([]*domain.CorpAction, error) {
	var models []CorpActionModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("ex_date DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	actions := make([]*domain.CorpAction, 0, len(models))
	for i := range models {
		actions = append(actions, actionToDomain(&models[i]))
	}
	return actions, nil
}

func (r *corpActionRepo) ListPendingActions(ctx context.Context, symbol string) ([]*domain.CorpAction, error) {
	query := r.db.WithContext(ctx).Where("status = ?", "ANNOUNCED")
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}

	var models []CorpActionModel
	if err := query.Order("ex_date ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	actions := make([]*domain.CorpAction, 0, len(models))
	for i := range models {
		actions = append(actions, actionToDomain(&models[i]))
	}
	return actions, nil
}

func (r *corpActionRepo) SaveExecution(ctx context.Context, execution *domain.CorpActionExecution) error {
	return r.db.WithContext(ctx).Create(executionToModel(execution)).Error
}

func mapAssetVaults(models []AssetVaultModel) []*domain.AssetVault {
	vaults := make([]*domain.AssetVault, 0, len(models))
	for i := range models {
		vaults = append(vaults, assetVaultToDomain(&models[i]))
	}
	return vaults
}

func assetVaultToDomain(model *AssetVaultModel) *domain.AssetVault {
	if model == nil {
		return nil
	}
	return &domain.AssetVault{
		VaultID:   model.VaultID,
		Type:      domain.VaultType(model.Type),
		UserID:    model.UserID,
		Symbol:    model.Symbol,
		Balance:   model.Balance,
		Locked:    model.Locked,
		UpdatedAt: model.UpdatedAt,
	}
}

func domainVaultToModel(vault *domain.AssetVault) *AssetVaultModel {
	return &AssetVaultModel{
		VaultID: vault.VaultID,
		Type:    string(vault.Type),
		UserID:  vault.UserID,
		Symbol:  vault.Symbol,
		Balance: vault.Balance,
		Locked:  vault.Locked,
	}
}

func transferToDomain(model *CustodyTransferModel) *domain.CustodyTransfer {
	if model == nil {
		return nil
	}
	return &domain.CustodyTransfer{
		TransferID: model.TransferID,
		FromVault:  model.FromVault,
		ToVault:    model.ToVault,
		Symbol:     model.Symbol,
		Amount:     model.Amount,
		Reason:     model.Reason,
		Timestamp:  model.Timestamp,
	}
}

func domainTransferToModel(transfer *domain.CustodyTransfer) *CustodyTransferModel {
	return &CustodyTransferModel{
		TransferID: transfer.TransferID,
		FromVault:  transfer.FromVault,
		ToVault:    transfer.ToVault,
		Symbol:     transfer.Symbol,
		Amount:     transfer.Amount,
		Reason:     transfer.Reason,
		Timestamp:  transfer.Timestamp,
	}
}

func actionToDomain(model *CorpActionModel) *domain.CorpAction {
	if model == nil {
		return nil
	}
	return &domain.CorpAction{
		ActionID:   model.ActionID,
		Symbol:     model.Symbol,
		Type:       domain.CorpActionType(model.Type),
		Ratio:      model.Ratio,
		RecordDate: model.RecordDate,
		ExDate:     model.ExDate,
		PayDate:    model.PayDate,
		Status:     model.Status,
	}
}

func domainActionToModel(action *domain.CorpAction) *CorpActionModel {
	return &CorpActionModel{
		ActionID:   action.ActionID,
		Symbol:     action.Symbol,
		Type:       string(action.Type),
		Ratio:      action.Ratio,
		RecordDate: action.RecordDate,
		ExDate:     action.ExDate,
		PayDate:    action.PayDate,
		Status:     action.Status,
	}
}

func executionToModel(execution *domain.CorpActionExecution) *CorpActionExecutionModel {
	return &CorpActionExecutionModel{
		ExecutionID: execution.ExecutionID,
		ActionID:    execution.ActionID,
		UserID:      execution.UserID,
		OldPosition: execution.OldPosition,
		NewPosition: execution.NewPosition,
		ChangeAmt:   execution.ChangeAmt,
		ExecutedAt:  execution.ExecutedAt,
	}
}
