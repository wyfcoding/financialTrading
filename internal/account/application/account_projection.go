package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

// AccountProjectionService 负责将账户事件投影到读模型。
type AccountProjectionService struct {
	repo     domain.AccountRepository
	readRepo domain.AccountReadRepository
	logger   *slog.Logger
}

func NewAccountProjectionService(repo domain.AccountRepository, readRepo domain.AccountReadRepository, logger *slog.Logger) *AccountProjectionService {
	return &AccountProjectionService{
		repo:     repo,
		readRepo: readRepo,
		logger:   logger,
	}
}

func (s *AccountProjectionService) Refresh(ctx context.Context, accountID string) error {
	if s.readRepo == nil {
		return nil
	}
	account, err := s.repo.Get(ctx, accountID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load account for projection", "account_id", accountID, "error", err)
		return err
	}
	if account == nil {
		_ = s.readRepo.Delete(ctx, accountID)
		return nil
	}
	if err := s.readRepo.Save(ctx, account); err != nil {
		s.logger.ErrorContext(ctx, "failed to save account cache", "account_id", accountID, "error", err)
		return err
	}
	return nil
}
