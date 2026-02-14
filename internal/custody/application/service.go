package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/custody/domain"
)

type TransferInternalCommand struct {
	FromVault string
	ToVault   string
	Symbol    string
	Amount    int64
	Reason    string
}

type SegregateCommand struct {
	UserID    uint64
	Symbol    string
	Amount    int64
	FromVault string
}

type AnnounceActionCommand struct {
	Symbol     string
	Type       domain.CorpActionType
	Ratio      float64
	RecordDate time.Time
	ExDate     time.Time
	PayDate    time.Time
}

type CustodyCommandService struct {
	vaultRepo  domain.CustodyRepository
	actionRepo domain.CorpActionRepository
	logger     *slog.Logger
}

func NewCustodyCommandService(
	vaultRepo domain.CustodyRepository,
	actionRepo domain.CorpActionRepository,
	logger *slog.Logger,
) *CustodyCommandService {
	return &CustodyCommandService{
		vaultRepo:  vaultRepo,
		actionRepo: actionRepo,
		logger:     logger,
	}
}

func (s *CustodyCommandService) TransferInternal(ctx context.Context, cmd TransferInternalCommand) (string, error) {
	s.logger.Info("processing internal transfer", "from", cmd.FromVault, "to", cmd.ToVault, "amount", cmd.Amount)

	fromVault, err := s.vaultRepo.FindVaultByID(ctx, cmd.FromVault)
	if err != nil {
		return "", fmt.Errorf("source vault not found: %w", err)
	}
	toVault, err := s.vaultRepo.FindVaultByID(ctx, cmd.ToVault)
	if err != nil {
		return "", fmt.Errorf("destination vault not found: %w", err)
	}

	if fromVault.Symbol != toVault.Symbol {
		return "", fmt.Errorf("symbol mismatch: %s vs %s", fromVault.Symbol, toVault.Symbol)
	}

	if fromVault.Symbol != cmd.Symbol {
		return "", fmt.Errorf("command symbol mismatch")
	}

	if err := fromVault.SafeDebit(cmd.Amount); err != nil {
		return "", err
	}
	toVault.SafeCredit(cmd.Amount)

	transferID := fmt.Sprintf("TX-%d", time.Now().UnixNano())
	transfer := &domain.CustodyTransfer{
		TransferID: transferID,
		FromVault:  cmd.FromVault,
		ToVault:    cmd.ToVault,
		Symbol:     cmd.Symbol,
		Amount:     cmd.Amount,
		Reason:     cmd.Reason,
		Timestamp:  time.Now(),
	}

	if err := s.vaultRepo.SaveVault(ctx, fromVault); err != nil {
		return "", fmt.Errorf("save source vault: %w", err)
	}
	if err := s.vaultRepo.SaveVault(ctx, toVault); err != nil {
		return "", fmt.Errorf("save destination vault: %w", err)
	}
	if err := s.vaultRepo.SaveTransfer(ctx, transfer); err != nil {
		return "", fmt.Errorf("save transfer record: %w", err)
	}

	s.logger.Info("internal transfer completed", "transfer_id", transferID, "amount", cmd.Amount)
	return transferID, nil
}

func (s *CustodyCommandService) Segregate(ctx context.Context, cmd SegregateCommand) error {
	s.logger.Info("enforcing asset segregation", "user_id", cmd.UserID, "symbol", cmd.Symbol)

	omnibusVault, err := s.vaultRepo.FindVaultByType(ctx, domain.VaultOmnibus, cmd.Symbol)
	if err != nil {
		return fmt.Errorf("omnibus vault not found: %w", err)
	}

	customerVault, err := s.vaultRepo.FindVaultByUser(ctx, cmd.UserID, cmd.Symbol)
	if err != nil {
		customerVault = domain.NewCustomerVault(cmd.UserID, cmd.Symbol)
	}

	if omnibusVault.AvailableBalance() < cmd.Amount {
		return fmt.Errorf("insufficient balance in omnibus vault")
	}

	if err := omnibusVault.SafeDebit(cmd.Amount); err != nil {
		return err
	}
	customerVault.SafeCredit(cmd.Amount)

	transferID := fmt.Sprintf("SEG-%d", time.Now().UnixNano())
	transfer := &domain.CustodyTransfer{
		TransferID: transferID,
		FromVault:  omnibusVault.VaultID,
		ToVault:    customerVault.VaultID,
		Symbol:     cmd.Symbol,
		Amount:     cmd.Amount,
		Reason:     "asset_segregation",
		Timestamp:  time.Now(),
	}

	if err := s.vaultRepo.SaveVault(ctx, omnibusVault); err != nil {
		return err
	}
	if err := s.vaultRepo.SaveVault(ctx, customerVault); err != nil {
		return err
	}
	if err := s.vaultRepo.SaveTransfer(ctx, transfer); err != nil {
		return err
	}

	s.logger.Info("asset segregation completed",
		"user_id", cmd.UserID,
		"symbol", cmd.Symbol,
		"amount", cmd.Amount,
		"transfer_id", transferID,
	)

	return nil
}

func (s *CustodyCommandService) SegregateAllUserAssets(ctx context.Context, userID uint64) error {
	s.logger.Info("segregating all user assets", "user_id", userID)

	omnibusVaults, err := s.vaultRepo.ListVaultsByType(ctx, domain.VaultOmnibus)
	if err != nil {
		return err
	}

	for _, omnibusVault := range omnibusVaults {
		if omnibusVault.AvailableBalance() == 0 {
			continue
		}

		customerVault, err := s.vaultRepo.FindVaultByUser(ctx, userID, omnibusVault.Symbol)
		if err != nil {
			customerVault = domain.NewCustomerVault(userID, omnibusVault.Symbol)
		}

		amount := omnibusVault.AvailableBalance()
		if err := omnibusVault.SafeDebit(amount); err != nil {
			continue
		}
		customerVault.SafeCredit(amount)

		transferID := fmt.Sprintf("SEG-%d", time.Now().UnixNano())
		transfer := &domain.CustodyTransfer{
			TransferID: transferID,
			FromVault:  omnibusVault.VaultID,
			ToVault:    customerVault.VaultID,
			Symbol:     omnibusVault.Symbol,
			Amount:     amount,
			Reason:     "full_asset_segregation",
			Timestamp:  time.Now(),
		}

		_ = s.vaultRepo.SaveVault(ctx, omnibusVault)
		_ = s.vaultRepo.SaveVault(ctx, customerVault)
		_ = s.vaultRepo.SaveTransfer(ctx, transfer)

		s.logger.Info("segregated asset",
			"user_id", userID,
			"symbol", omnibusVault.Symbol,
			"amount", amount,
		)
	}

	return nil
}

func (s *CustodyCommandService) AnnounceAction(ctx context.Context, cmd AnnounceActionCommand) (string, error) {
	actionID := fmt.Sprintf("CA-%s-%d", cmd.Symbol, time.Now().UnixNano())
	action := &domain.CorpAction{
		ActionID:   actionID,
		Symbol:     cmd.Symbol,
		Type:       cmd.Type,
		Ratio:      cmd.Ratio,
		RecordDate: cmd.RecordDate,
		ExDate:     cmd.ExDate,
		PayDate:    cmd.PayDate,
		Status:     "ANNOUNCED",
	}

	if err := s.actionRepo.SaveAction(ctx, action); err != nil {
		return "", err
	}

	s.logger.Info("corporate action announced",
		"action_id", actionID,
		"symbol", cmd.Symbol,
		"type", cmd.Type,
	)

	return actionID, nil
}

func (s *CustodyCommandService) ExecuteBatchAction(ctx context.Context, actionID string) (int, error) {
	action, err := s.actionRepo.FindActionByID(ctx, actionID)
	if err != nil {
		return 0, err
	}

	if action.Status != "ANNOUNCED" {
		return 0, fmt.Errorf("action not in announced state")
	}

	if time.Now().Before(action.ExDate) {
		return 0, fmt.Errorf("action not yet effective")
	}

	vaults, err := s.vaultRepo.ListVaultsBySymbol(ctx, action.Symbol)
	if err != nil {
		return 0, err
	}

	processedCount := 0
	for _, vault := range vaults {
		if vault.Type == domain.VaultHouse {
			continue
		}

		originalBalance := vault.Balance
		newBalance := int64(float64(originalBalance) * action.Ratio)
		adjustment := newBalance - originalBalance

		if adjustment > 0 {
			vault.SafeCredit(adjustment)
		} else if adjustment < 0 {
			_ = vault.SafeDebit(-adjustment)
		}

		if err := s.vaultRepo.SaveVault(ctx, vault); err != nil {
			s.logger.Warn("failed to process vault for corporate action",
				"vault_id", vault.VaultID,
				"action_id", actionID,
				"error", err,
			)
			continue
		}

		processedCount++
	}

	action.Status = "EXECUTED"
	_ = s.actionRepo.SaveAction(ctx, action)

	s.logger.Info("corporate action executed",
		"action_id", actionID,
		"symbol", action.Symbol,
		"vaults_processed", processedCount,
	)

	return processedCount, nil
}

func (s *CustodyCommandService) LockVault(ctx context.Context, vaultID string, amount int64, reason string) error {
	vault, err := s.vaultRepo.FindVaultByID(ctx, vaultID)
	if err != nil {
		return err
	}

	if err := vault.Lock(amount); err != nil {
		return err
	}

	return s.vaultRepo.SaveVault(ctx, vault)
}

func (s *CustodyCommandService) UnlockVault(ctx context.Context, vaultID string, amount int64) error {
	vault, err := s.vaultRepo.FindVaultByID(ctx, vaultID)
	if err != nil {
		return err
	}

	vault.Unlock(amount)
	return s.vaultRepo.SaveVault(ctx, vault)
}

type CustodyQueryService struct {
	vaultRepo  domain.CustodyRepository
	actionRepo domain.CorpActionRepository
	logger     *slog.Logger
}

func NewCustodyQueryService(
	vaultRepo domain.CustodyRepository,
	actionRepo domain.CorpActionRepository,
	logger *slog.Logger,
) *CustodyQueryService {
	return &CustodyQueryService{
		vaultRepo:  vaultRepo,
		actionRepo: actionRepo,
		logger:     logger,
	}
}

func (s *CustodyQueryService) GetHolding(ctx context.Context, vaultID string) (*domain.AssetVault, error) {
	return s.vaultRepo.FindVaultByID(ctx, vaultID)
}

func (s *CustodyQueryService) GetUserHoldings(ctx context.Context, userID uint64) ([]*domain.AssetVault, error) {
	return s.vaultRepo.ListVaultsByUser(ctx, userID)
}

func (s *CustodyQueryService) GetTransferHistory(ctx context.Context, vaultID string, limit int) ([]*domain.CustodyTransfer, error) {
	return s.vaultRepo.ListTransfersByVault(ctx, vaultID, limit)
}

func (s *CustodyQueryService) GetCorpAction(ctx context.Context, actionID string) (*domain.CorpAction, error) {
	return s.actionRepo.FindActionByID(ctx, actionID)
}

func (s *CustodyQueryService) ListPendingCorpActions(ctx context.Context, symbol string) ([]*domain.CorpAction, error) {
	return s.actionRepo.ListPendingActions(ctx, symbol)
}

func (s *CustodyQueryService) GetVaultSummary(ctx context.Context, vaultType domain.VaultType) (*VaultSummary, error) {
	vaults, err := s.vaultRepo.ListVaultsByType(ctx, vaultType)
	if err != nil {
		return nil, err
	}

	summary := &VaultSummary{
		VaultType: string(vaultType),
		BySymbol:  make(map[string]int64),
	}

	for _, v := range vaults {
		summary.TotalVaults++
		summary.TotalBalance += v.Balance
		summary.TotalLocked += v.Locked
		summary.BySymbol[v.Symbol] += v.Balance
	}

	return summary, nil
}

type VaultSummary struct {
	VaultType    string
	TotalVaults  int
	TotalBalance int64
	TotalLocked  int64
	BySymbol     map[string]int64
}

type CustodyAppService struct {
	cmd   *CustodyCommandService
	query *CustodyQueryService
}

func NewCustodyAppService(
	vaultRepo domain.CustodyRepository,
	actionRepo domain.CorpActionRepository,
	logger *slog.Logger,
) *CustodyAppService {
	return &CustodyAppService{
		cmd:   NewCustodyCommandService(vaultRepo, actionRepo, logger),
		query: NewCustodyQueryService(vaultRepo, actionRepo, logger),
	}
}

func (s *CustodyAppService) TransferInternal(ctx context.Context, fromVault, toVault, symbol string, amount int64, reason string) (string, error) {
	return s.cmd.TransferInternal(ctx, TransferInternalCommand{
		FromVault: fromVault,
		ToVault:   toVault,
		Symbol:    symbol,
		Amount:    amount,
		Reason:    reason,
	})
}

func (s *CustodyAppService) Segregate(ctx context.Context, userID uint64) error {
	return s.cmd.SegregateAllUserAssets(ctx, userID)
}

func (s *CustodyAppService) GetHolding(ctx context.Context, vaultID string) (*domain.AssetVault, error) {
	return s.query.GetHolding(ctx, vaultID)
}

func (s *CustodyAppService) AnnounceAction(ctx context.Context, symbol, actionType string, ratio float64, recordDate, exDate, payDate time.Time) (string, error) {
	return s.cmd.AnnounceAction(ctx, AnnounceActionCommand{
		Symbol:     symbol,
		Type:       domain.CorpActionType(actionType),
		Ratio:      ratio,
		RecordDate: recordDate,
		ExDate:     exDate,
		PayDate:    payDate,
	})
}

func (s *CustodyAppService) ExecuteBatchAction(ctx context.Context, actionID string) error {
	_, err := s.cmd.ExecuteBatchAction(ctx, actionID)
	return err
}
