package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
)

// CreateAccountCommand 开户命令
type CreateAccountCommand struct {
	UserID      string
	AccountType string
	Currency    string
}

// DepositCommand 充值命令
type DepositCommand struct {
	AccountID string
	Amount    decimal.Decimal
}

// FreezeCommand 冻结命令
type FreezeCommand struct {
	AccountID string
	Amount    decimal.Decimal
	Reason    string
}

// AccountCommandService 处理账户相关的写操作。
type AccountCommandService struct {
	repo       domain.AccountRepository
	eventStore domain.EventStore
	publisher  domain.EventPublisher
	logger     *slog.Logger
}

func NewAccountCommandService(
	repo domain.AccountRepository,
	eventStore domain.EventStore,
	publisher domain.EventPublisher,
	logger *slog.Logger,
) *AccountCommandService {
	return &AccountCommandService{
		repo:       repo,
		eventStore: eventStore,
		publisher:  publisher,
		logger:     logger,
	}
}

// CreateAccount 处理开户
func (s *AccountCommandService) CreateAccount(ctx context.Context, cmd CreateAccountCommand) (*AccountDTO, error) {
	// 生成 ID (在应用层生成符合 Clean Architecture)
	accountID := fmt.Sprintf("ACC-%d", idgen.GenID())

	// 创建领域对象
	account := domain.NewAccount(
		accountID,
		cmd.UserID,
		cmd.Currency,
		domain.AccountType(cmd.AccountType),
	)

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Save(txCtx, account); err != nil {
			return err
		}
		if err := s.eventStore.Save(txCtx, account.AccountID, account.GetUncommittedEvents(), account.Version()); err != nil {
			return err
		}
		account.MarkCommitted()

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountCreatedEventType, accountID, map[string]any{
			"account_id": accountID,
			"user_id":    cmd.UserID,
			"currency":   cmd.Currency,
		})
	})

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create account", "error", err)
		return nil, err
	}

	return s.toDTO(account), nil
}

// Deposit 处理充值
func (s *AccountCommandService) Deposit(ctx context.Context, cmd DepositCommand) error {
	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		account, err := s.repo.Get(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}
		if account == nil {
			return fmt.Errorf("account not found: %s", cmd.AccountID)
		}

		account.Deposit(cmd.Amount)

		if err := s.repo.Save(txCtx, account); err != nil {
			return err
		}
		if err := s.eventStore.Save(txCtx, account.AccountID, account.GetUncommittedEvents(), account.Version()); err != nil {
			return err
		}
		account.MarkCommitted()

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountDepositedEventType, fmt.Sprintf("DEP-%d", idgen.GenID()), map[string]any{
			"account_id": account.AccountID,
			"amount":     cmd.Amount.String(),
			"balance":    account.Balance.String(),
		})
	})
}

func (s *AccountCommandService) toDTO(a *domain.Account) *AccountDTO {
	return &AccountDTO{
		AccountID:        a.AccountID,
		UserID:           a.UserID,
		AccountType:      string(a.AccountType),
		Currency:         a.Currency,
		Balance:          a.Balance.String(),
		AvailableBalance: a.AvailableBalance.String(),
		FrozenBalance:    a.FrozenBalance.String(),
		CreatedAt:        a.CreatedAt.Unix(),
		UpdatedAt:        a.UpdatedAt.Unix(),
		Version:          a.Version(),
	}
}

// Freeze 处理冻结
func (s *AccountCommandService) Freeze(ctx context.Context, cmd FreezeCommand) error {
	return s.repo.WithTx(ctx, func(txCtx context.Context) error {
		account, err := s.repo.Get(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}
		if account == nil {
			return fmt.Errorf("account not found")
		}

		if success := account.Freeze(cmd.Amount, cmd.Reason); !success {
			return fmt.Errorf("insufficient available balance")
		}

		if err := s.repo.Save(txCtx, account); err != nil {
			return err
		}
		if err := s.eventStore.Save(txCtx, account.AccountID, account.GetUncommittedEvents(), account.Version()); err != nil {
			return err
		}
		account.MarkCommitted()

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountFrozenEventType, fmt.Sprintf("FRZ-%d", idgen.GenID()), map[string]any{
			"account_id": account.AccountID,
			"amount":     cmd.Amount.String(),
			"reason":     cmd.Reason,
		})
	})
}

// SagaDeductFrozen Saga 接口: 从冻结金额中扣除
func (s *AccountCommandService) SagaDeductFrozen(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		accounts, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		if success := targetAccount.DeductFrozen(amount); !success {
			return fmt.Errorf("insufficient frozen balance")
		}

		if err := s.repo.Save(txCtx, targetAccount); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountDeductedEventType, fmt.Sprintf("SAGA-DED-%s", userID), map[string]any{
			"type":       "SAGA_DEDUCT",
			"account_id": targetAccount.AccountID,
			"user_id":    userID,
			"amount":     amount.String(),
		})
	})
}

// SagaRefundFrozen Saga 补偿接口: 退回冻结金额
func (s *AccountCommandService) SagaRefundFrozen(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		accounts, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		if success := targetAccount.Unfreeze(amount); !success {
			return fmt.Errorf("failed to unfreeze for refund")
		}

		if err := s.repo.Save(txCtx, targetAccount); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountUnfrozenEventType, fmt.Sprintf("SAGA-REF-%s", userID), map[string]any{
			"type":       "SAGA_REFUND",
			"account_id": targetAccount.AccountID,
			"user_id":    userID,
			"amount":     amount.String(),
		})
	})
}

// SagaAddBalance Saga 接口: 增加余额
func (s *AccountCommandService) SagaAddBalance(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		accounts, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		targetAccount.Deposit(amount)
		if err := s.repo.Save(txCtx, targetAccount); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountDepositedEventType, fmt.Sprintf("SAGA-ADD-%s", userID), map[string]any{
			"type":       "SAGA_ADD",
			"account_id": targetAccount.AccountID,
			"user_id":    userID,
			"amount":     amount.String(),
		})
	})
}

// SagaSubBalance Saga 补偿接口: 扣减余额
func (s *AccountCommandService) SagaSubBalance(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		accounts, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		if success := targetAccount.Withdraw(amount); !success {
			return fmt.Errorf("insufficient balance for compensation")
		}
		if err := s.repo.Save(txCtx, targetAccount); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountWithdrawnEventType, fmt.Sprintf("SAGA-SUB-%s", userID), map[string]any{
			"type":       "SAGA_SUB",
			"account_id": targetAccount.AccountID,
			"user_id":    userID,
			"amount":     amount.String(),
		})
	})
}

// TccTryFreeze TCC Try: 尝试冻结
func (s *AccountCommandService) TccTryFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		accounts, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		if success := targetAccount.Freeze(amount, "TCC Freeze"); !success {
			return fmt.Errorf("insufficient balance for TCC freeze")
		}
		if err := s.repo.Save(txCtx, targetAccount); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountFrozenEventType, fmt.Sprintf("TCC-TRY-%s", userID), map[string]any{
			"type":       "TCC_TRY",
			"account_id": targetAccount.AccountID,
			"user_id":    userID,
			"amount":     amount.String(),
		})
	})
}

// TccConfirmFreeze TCC Confirm: 确认冻结
func (s *AccountCommandService) TccConfirmFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return nil
	})
}

// TccCancelFreeze TCC Cancel: 取消冻结
func (s *AccountCommandService) TccCancelFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(txCtx context.Context) error {
		accounts, err := s.repo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}

		var targetAccount *domain.Account
		for _, acc := range accounts {
			if acc.Currency == currency {
				targetAccount = acc
				break
			}
		}
		if targetAccount == nil {
			return fmt.Errorf("account not found for user %s currency %s", userID, currency)
		}

		if success := targetAccount.Unfreeze(amount); !success {
			return fmt.Errorf("failed to unfreeze for TCC cancel")
		}
		if err := s.repo.Save(txCtx, targetAccount); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		tx := contextx.GetTx(txCtx)
		return s.publisher.PublishInTx(ctx, tx, domain.AccountUnfrozenEventType, fmt.Sprintf("TCC-CANCEL-%s", userID), map[string]any{
			"type":       "TCC_CANCEL",
			"account_id": targetAccount.AccountID,
			"user_id":    userID,
			"amount":     amount.String(),
		})
	})
}
