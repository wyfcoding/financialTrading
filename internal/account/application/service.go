package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

// AccountService 应用层服务，负责协调领域对象与基础设施
type AccountService struct {
	repo       domain.AccountRepository
	eventStore domain.EventStore
	outbox     *outbox.Manager
	db         *gorm.DB // 用于开启事务 (Infrastructure Leak, but pragmatic for Go)
}

func NewAccountService(
	repo domain.AccountRepository,
	eventStore domain.EventStore,
	outbox *outbox.Manager,
	db *gorm.DB,
) *AccountService {
	return &AccountService{
		repo:       repo,
		eventStore: eventStore,
		outbox:     outbox,
		db:         db,
	}
}

// CreateAccount 处理开户
func (s *AccountService) CreateAccount(ctx context.Context, cmd CreateAccountCommand) (*AccountDTO, error) {
	// 生成 ID (在应用层生成符合 Clean Architecture)
	accountID := fmt.Sprintf("ACC-%d", idgen.GenID())

	// 创建领域对象
	account := domain.NewAccount(
		accountID,
		cmd.UserID,
		cmd.Currency,
		domain.AccountType(cmd.AccountType),
	)

	// 生成领域事件
	event := domain.AccountCreatedEvent{
		BaseEvent:   domain.BaseEvent{Timestamp: account.CreatedAt},
		AccountID:   account.ID,
		UserID:      account.UserID,
		AccountType: string(account.AccountType),
		Currency:    account.Currency,
	}

	// 事务保存
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx) // 传递事务上下文

		if err := s.repo.Save(txCtx, account); err != nil {
			return err
		}

		if err := s.eventStore.Append(txCtx, account.ID, []domain.AccountEvent{event}); err != nil {
			return err
		}

		// 发送集成事件 (Outbox Pattern)
		return s.outbox.PublishInTx(ctx, tx, "account.created", accountID, map[string]any{
			"account_id": accountID, "user_id": cmd.UserID, "currency": cmd.Currency,
		})
	})

	if err != nil {
		slog.ErrorContext(ctx, "failed to create account", "error", err)
		return nil, err
	}

	return s.toDTO(account), nil
}

// Deposit 处理充值
func (s *AccountService) Deposit(ctx context.Context, cmd DepositCommand) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)

		// 1. Load
		account, err := s.repo.Get(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}
		if account == nil {
			return fmt.Errorf("account not found")
		}

		// 2. Do Domain Logic
		account.Deposit(cmd.Amount)

		// Event
		event := domain.FundsDepositedEvent{
			BaseEvent: domain.BaseEvent{Timestamp: account.UpdatedAt},
			AccountID: account.ID,
			Amount:    cmd.Amount,
			Balance:   account.Balance,
		}

		// 3. Save
		if err := s.repo.Save(txCtx, account); err != nil {
			return err
		}
		if err := s.eventStore.Append(txCtx, account.ID, []domain.AccountEvent{event}); err != nil {
			return err
		}

		// Integration Event
		return s.outbox.PublishInTx(ctx, tx, "account.deposited", fmt.Sprintf("DEP-%d", idgen.GenID()), map[string]any{
			"account_id": account.ID, "amount": cmd.Amount.String(), "balance": account.Balance.String(),
		})
	})
}

// Freeze 处理冻结
func (s *AccountService) Freeze(ctx context.Context, cmd FreezeCommand) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)

		account, err := s.repo.Get(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}
		if account == nil {
			return fmt.Errorf("account not found")
		}

		if success := account.Freeze(cmd.Amount); !success {
			return fmt.Errorf("insufficient available balance")
		}

		event := domain.FundsFrozenEvent{
			BaseEvent: domain.BaseEvent{Timestamp: account.UpdatedAt},
			AccountID: account.ID,
			Amount:    cmd.Amount,
			Reason:    cmd.Reason,
		}

		if err := s.repo.Save(txCtx, account); err != nil {
			return err
		}
		return s.eventStore.Append(txCtx, account.ID, []domain.AccountEvent{event})
	})
}

// SagaDeductFrozen Saga 接口: 从冻结金额中扣除
func (s *AccountService) SagaDeductFrozen(ctx context.Context, barrier any, userID, currency string, amountString string) error {
	amount, _ := decimal.NewFromString(amountString)

	// 使用 Repo 的 Barrier 支持
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		accounts, err := s.repo.GetByUserID(ctx, userID)
		if err != nil {
			return err
		}

		// 查找对应币种账户
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

		// Domain Logic
		if success := targetAccount.DeductFrozen(amount); !success {
			return fmt.Errorf("insufficient frozen balance")
		}

		// Save
		if err := s.repo.Save(ctx, targetAccount); err != nil {
			return err
		}

		// Outbox Event inside barrier transaction
		tx, _ := contextx.GetTx(ctx).(*gorm.DB)
		return s.outbox.PublishInTx(ctx, tx, "account.deducted", fmt.Sprintf("SAGA-DED-%s", userID), map[string]any{
			"type": "SAGA_DEDUCT", "user_id": userID, "amount": amountString,
		})
	})
}

func (s *AccountService) toDTO(a *domain.Account) *AccountDTO {
	return &AccountDTO{
		AccountID:        a.ID,
		UserID:           a.UserID,
		AccountType:      string(a.AccountType),
		Currency:         a.Currency,
		Balance:          a.Balance.String(),
		AvailableBalance: a.AvailableBalance.String(),
		FrozenBalance:    a.FrozenBalance.String(),
		UpdatedAt:        a.UpdatedAt.Unix(),
		Version:          a.Version,
	}
}
