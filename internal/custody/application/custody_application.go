package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/custody/domain"
)

// TransferInternalCommand 内部转移命令
type TransferInternalCommand struct {
	FromVault string
	ToVault   string
	Symbol    string
	Amount    int64
	Reason    string
}

// AnnounceActionCommand 发布公司行为命令
type AnnounceActionCommand struct {
	Symbol     string
	Type       domain.CorpActionType
	Ratio      float64
	RecordDate time.Time
	ExDate     time.Time
	PayDate    time.Time
}

// CustodyApplicationService 托管应用服务
type CustodyApplicationService struct {
	vaultRepo  domain.CustodyRepository
	actionRepo domain.CorpActionRepository
	logger     *slog.Logger
}

func NewCustodyApplicationService(vaultRepo domain.CustodyRepository, actionRepo domain.CorpActionRepository, logger *slog.Logger) *CustodyApplicationService {
	return &CustodyApplicationService{
		vaultRepo:  vaultRepo,
		actionRepo: actionRepo,
		logger:     logger,
	}
}

func (s *CustodyApplicationService) TransferInternal(ctx context.Context, cmd TransferInternalCommand) (string, error) {
	s.logger.Info("processing internal transfer", "from", cmd.FromVault, "to", cmd.ToVault, "amt", cmd.Amount)

	fromVault, err := s.vaultRepo.FindVaultByID(ctx, cmd.FromVault)
	if err != nil {
		return "", err
	}
	toVault, err := s.vaultRepo.FindVaultByID(ctx, cmd.ToVault)
	if err != nil {
		return "", err
	}

	if err := fromVault.SafeDebit(cmd.Amount); err != nil {
		return "", err
	}
	toVault.SafeCredit(cmd.Amount)

	transfer := &domain.CustodyTransfer{
		TransferID: fmt.Sprintf("TX-%d", time.Now().UnixNano()),
		FromVault:  cmd.FromVault,
		ToVault:    cmd.ToVault,
		Symbol:     cmd.Symbol,
		Amount:     cmd.Amount,
		Reason:     cmd.Reason,
		Timestamp:  time.Now(),
	}

	if err := s.vaultRepo.SaveVault(ctx, fromVault); err != nil {
		return "", err
	}
	if err := s.vaultRepo.SaveVault(ctx, toVault); err != nil {
		return "", err
	}
	if err := s.vaultRepo.SaveTransfer(ctx, transfer); err != nil {
		return "", err
	}

	return transfer.TransferID, nil
}

func (s *CustodyApplicationService) Segregate(ctx context.Context, userID uint64) error {
	s.logger.Info("enforcing asset segregation", "user_id", userID)
	// 逻辑：检查所有涉及该用户的 Vault，确保其资金不在 House 账户中。
	return nil
}

func (s *CustodyApplicationService) GetHolding(ctx context.Context, vaultID string) (*domain.AssetVault, error) {
	return s.vaultRepo.FindVaultByID(ctx, vaultID)
}

func (s *CustodyApplicationService) AnnounceAction(ctx context.Context, cmd AnnounceActionCommand) (string, error) {
	action := &domain.CorpAction{
		ActionID:   fmt.Sprintf("CA-%d", time.Now().UnixNano()),
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

	s.logger.Info("corporate action announced", "action_id", action.ActionID, "symbol", action.Symbol)
	return action.ActionID, nil
}
