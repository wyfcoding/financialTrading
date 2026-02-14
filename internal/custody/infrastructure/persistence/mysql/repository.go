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

func (r *CustodyRepo) FindVaultByUser(ctx context.Context, userID uint64, symbol string) (*domain.AssetVault, error) {
	var model AssetVaultModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol = ? AND type = ?", userID, symbol, string(domain.VaultCustomer)).
		First(&model).Error; err != nil {
		return nil, err
	}
	return toDomainVault(&model), nil
}

func (r *CustodyRepo) FindVaultByType(ctx context.Context, vaultType domain.VaultType, symbol string) (*domain.AssetVault, error) {
	var model AssetVaultModel
	query := r.db.WithContext(ctx).Where("type = ?", string(vaultType))
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	if err := query.First(&model).Error; err != nil {
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

func (r *CustodyRepo) ListVaultsByUser(ctx context.Context, userID uint64) ([]*domain.AssetVault, error) {
	return r.FindVaultsByUserID(ctx, userID)
}

func (r *CustodyRepo) ListVaultsByType(ctx context.Context, vaultType domain.VaultType) ([]*domain.AssetVault, error) {
	var models []AssetVaultModel
	if err := r.db.WithContext(ctx).Where("type = ?", string(vaultType)).Find(&models).Error; err != nil {
		return nil, err
	}
	vaults := make([]*domain.AssetVault, 0, len(models))
	for i := range models {
		vaults = append(vaults, toDomainVault(&models[i]))
	}
	return vaults, nil
}

func (r *CustodyRepo) ListVaultsBySymbol(ctx context.Context, symbol string) ([]*domain.AssetVault, error) {
	var models []AssetVaultModel
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Find(&models).Error; err != nil {
		return nil, err
	}
	vaults := make([]*domain.AssetVault, 0, len(models))
	for i := range models {
		vaults = append(vaults, toDomainVault(&models[i]))
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

func (r *CustodyRepo) ListTransfersByVault(ctx context.Context, vaultID string, limit int) ([]*domain.CustodyTransfer, error) {
	var models []CustodyTransferModel
	query := r.db.WithContext(ctx).
		Where("from_vault = ? OR to_vault = ?", vaultID, vaultID).
		Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	transfers := make([]*domain.CustodyTransfer, 0, len(models))
	for i := range models {
		transfers = append(transfers, toDomainTransfer(&models[i]))
	}
	return transfers, nil
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

func (r *CorpActionRepo) ListPendingActions(ctx context.Context, symbol string) ([]*domain.CorpAction, error) {
	var models []CorpActionModel
	query := r.db.WithContext(ctx).Where("status = ?", "ANNOUNCED")
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	actions := make([]*domain.CorpAction, 0, len(models))
	for i := range models {
		actions = append(actions, toDomainAction(&models[i]))
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

func toDomainTransfer(m *CustodyTransferModel) *domain.CustodyTransfer {
	if m == nil {
		return nil
	}
	return &domain.CustodyTransfer{
		TransferID: m.TransferID,
		FromVault:  m.FromVault,
		ToVault:    m.ToVault,
		Symbol:     m.Symbol,
		Amount:     m.Amount,
		Reason:     m.Reason,
		Timestamp:  m.Timestamp,
	}
}

var (
	_ domain.CustodyRepository    = (*CustodyRepo)(nil)
	_ domain.CorpActionRepository = (*CorpActionRepo)(nil)
)
