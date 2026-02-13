// Package application 资金服务应用层
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
	"github.com/wyfcoding/pkg/messagequeue"
)

// CommandService 资金命令服务
type CommandService struct {
	accountRepo    domain.AccountRepository
	txRepo         domain.TransactionRepository
	eventPublisher messagequeue.EventPublisher
	logger         *slog.Logger
}

// NewCommandService 创建命令服务
func NewCommandService(
	accountRepo domain.AccountRepository,
	txRepo domain.TransactionRepository,
	eventPublisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *CommandService {
	return &CommandService{
		accountRepo:    accountRepo,
		txRepo:         txRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateAccountCommand 创建账户命令
type CreateAccountCommand struct {
	OwnerID  uint64
	Type     domain.AccountType
	Currency domain.Currency
}

// CreateAccount 创建账户
func (s *CommandService) CreateAccount(ctx context.Context, cmd CreateAccountCommand) (uint64, error) {
	existing, _ := s.accountRepo.GetByOwner(ctx, cmd.OwnerID, cmd.Type, cmd.Currency)
	if existing != nil {
		return 0, fmt.Errorf("account already exists for owner %d type %d currency %d", cmd.OwnerID, cmd.Type, cmd.Currency)
	}

	account := domain.NewAccount(cmd.OwnerID, cmd.Type, cmd.Currency)

	if err := s.accountRepo.Save(ctx, account); err != nil {
		return 0, err
	}

	return uint64(account.ID), nil
}

// DepositCommand 充值命令
type DepositCommand struct {
	AccountID uint64
	Amount    int64
	RefID     string
	Source    string
}

// Deposit 充值
func (s *CommandService) Deposit(ctx context.Context, cmd DepositCommand) (string, error) {
	var txID string
	err := s.runInAccountTx(ctx, func(txCtx context.Context) error {
		account, err := s.accountRepo.GetWithLock(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}

		tx, err := account.Deposit(cmd.Amount, cmd.RefID, cmd.Source)
		if err != nil {
			return err
		}

		if err := s.saveAccountAndTx(txCtx, account, tx); err != nil {
			return err
		}
		txID = tx.TransactionID
		return nil
	})
	if err != nil {
		return "", err
	}

	return txID, nil
}

// FreezeCommand 冻结命令
type FreezeCommand struct {
	AccountID uint64
	Amount    int64
	RefID     string
	Reason    string
}

// Freeze 冻结
func (s *CommandService) Freeze(ctx context.Context, cmd FreezeCommand) (string, error) {
	var txID string
	err := s.runInAccountTx(ctx, func(txCtx context.Context) error {
		account, err := s.accountRepo.GetWithLock(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}

		tx, err := account.Freeze(cmd.Amount, cmd.RefID, cmd.Reason)
		if err != nil {
			return err
		}

		if err := s.saveAccountAndTx(txCtx, account, tx); err != nil {
			return err
		}
		txID = tx.TransactionID
		return nil
	})
	if err != nil {
		return "", err
	}

	return txID, nil
}

// UnfreezeCommand 解冻命令
type UnfreezeCommand struct {
	AccountID uint64
	Amount    int64
	RefID     string
	Reason    string
}

// Unfreeze 解冻
func (s *CommandService) Unfreeze(ctx context.Context, cmd UnfreezeCommand) (string, error) {
	var txID string
	err := s.runInAccountTx(ctx, func(txCtx context.Context) error {
		account, err := s.accountRepo.GetWithLock(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}

		tx, err := account.Unfreeze(cmd.Amount, cmd.RefID, cmd.Reason)
		if err != nil {
			return err
		}

		if err := s.saveAccountAndTx(txCtx, account, tx); err != nil {
			return err
		}
		txID = tx.TransactionID
		return nil
	})
	if err != nil {
		return "", err
	}

	return txID, nil
}

// DeductCommand 扣减命令
type DeductCommand struct {
	AccountID     uint64
	Amount        int64
	RefID         string
	Reason        string
	UnfreezeFirst bool
}

// Deduct 扣减
func (s *CommandService) Deduct(ctx context.Context, cmd DeductCommand) (string, error) {
	var txID string
	err := s.runInAccountTx(ctx, func(txCtx context.Context) error {
		account, err := s.accountRepo.GetWithLock(txCtx, cmd.AccountID)
		if err != nil {
			return err
		}

		tx, err := account.Deduct(cmd.Amount, cmd.UnfreezeFirst, cmd.RefID, cmd.Reason)
		if err != nil {
			return err
		}

		if err := s.saveAccountAndTx(txCtx, account, tx); err != nil {
			return err
		}
		txID = tx.TransactionID
		return nil
	})
	if err != nil {
		return "", err
	}

	return txID, nil
}

// TransferCommand 转账命令
type TransferCommand struct {
	FromAccountID uint64
	ToAccountID   uint64
	Amount        int64
	RefID         string
	Remark        string
}

// Transfer 转账（同库事务内原子完成双账户变更与双流水写入）
func (s *CommandService) Transfer(ctx context.Context, cmd TransferCommand) (string, error) {
	if cmd.FromAccountID == cmd.ToAccountID {
		return "", errors.New("cannot transfer to self")
	}
	if cmd.Amount <= 0 {
		return "", errors.New("amount must be positive")
	}

	// 简单的按ID顺序加锁避免死锁
	firstID, secondID := cmd.FromAccountID, cmd.ToAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	remark := cmd.Remark
	if remark == "" {
		remark = "account transfer"
	}

	transferRef := cmd.RefID
	err := s.runInAccountTx(ctx, func(txCtx context.Context) error {
		firstAccount, err := s.accountRepo.GetWithLock(txCtx, firstID)
		if err != nil {
			return err
		}
		secondAccount, err := s.accountRepo.GetWithLock(txCtx, secondID)
		if err != nil {
			return err
		}

		fromAccount, toAccount := firstAccount, secondAccount
		if cmd.FromAccountID != firstID {
			fromAccount, toAccount = secondAccount, firstAccount
		}
		if fromAccount == nil || toAccount == nil {
			return errors.New("account not found")
		}
		if fromAccount.Currency != toAccount.Currency {
			return fmt.Errorf("currency mismatch: from=%d to=%d", fromAccount.Currency, toAccount.Currency)
		}

		outTx, err := fromAccount.Deduct(cmd.Amount, false, transferRef, remark)
		if err != nil {
			return err
		}
		inTx, err := toAccount.Deposit(cmd.Amount, transferRef, remark)
		if err != nil {
			return err
		}

		if transferRef == "" {
			transferRef = outTx.TransactionID
		}
		outTx.Type = domain.TransactionTypeTransferOut
		outTx.ReferenceID = transferRef
		outTx.Remark = remark
		outTx.Amount = -cmd.Amount

		inTx.Type = domain.TransactionTypeTransferIn
		inTx.ReferenceID = transferRef
		inTx.Remark = remark
		inTx.Amount = cmd.Amount

		if err := s.accountRepo.Save(txCtx, fromAccount); err != nil {
			return err
		}
		if err := s.accountRepo.Save(txCtx, toAccount); err != nil {
			return err
		}
		if err := s.txRepo.Save(txCtx, outTx); err != nil {
			return err
		}
		if err := s.txRepo.Save(txCtx, inTx); err != nil {
			return err
		}

		s.publishEvents(txCtx, fromAccount.GetDomainEvents())
		s.publishEvents(txCtx, toAccount.GetDomainEvents())
		fromAccount.ClearDomainEvents()
		toAccount.ClearDomainEvents()
		return nil
	})
	if err != nil {
		return "", err
	}

	return transferRef, nil
}

func (s *CommandService) runInAccountTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	txRunner, ok := s.accountRepo.(interface {
		Transaction(ctx context.Context, fn func(ctx context.Context) error) error
	})
	if !ok {
		return errors.New("account repository does not support transaction")
	}
	return txRunner.Transaction(ctx, fn)
}

// saveAccountAndTx 辅助方法：保存账户和流水并发布事件
// 调用方应保证在数据库事务上下文中执行。
func (s *CommandService) saveAccountAndTx(ctx context.Context, account *domain.Account, tx *domain.Transaction) error {
	if err := s.accountRepo.Save(ctx, account); err != nil {
		return err
	}
	if err := s.txRepo.Save(ctx, tx); err != nil {
		// 严重错误：账户余额已扣减但流水保存失败
		s.logger.ErrorContext(ctx, "failed to save transaction after account update", "tx_id", tx.TransactionID, "error", err)
		return err
	}

	s.publishEvents(ctx, account.GetDomainEvents())
	account.ClearDomainEvents()
	return nil
}

// publishEvents 发布领域事件
func (s *CommandService) publishEvents(ctx context.Context, events []domain.DomainEvent) {
	for _, event := range events {
		if err := s.eventPublisher.Publish(ctx, event.EventName(), "", event); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish event",
				"event", event.EventName(),
				"error", err)
		}
	}
}
