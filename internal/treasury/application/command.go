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
	// 使用锁或乐观锁，这里简化为 GetWithLock (悲观锁)
	account, err := s.accountRepo.GetWithLock(ctx, cmd.AccountID)
	if err != nil {
		return "", err
	}

	tx, err := account.Deposit(cmd.Amount, cmd.RefID, cmd.Source)
	if err != nil {
		return "", err
	}

	if err := s.saveAccountAndTx(ctx, account, tx); err != nil {
		return "", err
	}

	return tx.TransactionID, nil
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
	account, err := s.accountRepo.GetWithLock(ctx, cmd.AccountID)
	if err != nil {
		return "", err
	}

	tx, err := account.Freeze(cmd.Amount, cmd.RefID, cmd.Reason)
	if err != nil {
		return "", err
	}

	if err := s.saveAccountAndTx(ctx, account, tx); err != nil {
		return "", err
	}

	return tx.TransactionID, nil
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
	account, err := s.accountRepo.GetWithLock(ctx, cmd.AccountID)
	if err != nil {
		return "", err
	}

	tx, err := account.Unfreeze(cmd.Amount, cmd.RefID, cmd.Reason)
	if err != nil {
		return "", err
	}

	if err := s.saveAccountAndTx(ctx, account, tx); err != nil {
		return "", err
	}

	return tx.TransactionID, nil
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
	account, err := s.accountRepo.GetWithLock(ctx, cmd.AccountID)
	if err != nil {
		return "", err
	}

	tx, err := account.Deduct(cmd.Amount, cmd.UnfreezeFirst, cmd.RefID, cmd.Reason)
	if err != nil {
		return "", err
	}

	if err := s.saveAccountAndTx(ctx, account, tx); err != nil {
		return "", err
	}

	return tx.TransactionID, nil
}

// TransferCommand 转账命令
type TransferCommand struct {
	FromAccountID uint64
	ToAccountID   uint64
	Amount        int64
	RefID         string
	Remark        string
}

// Transfer 转账 (简单实现，未涉及分布式事务，假设同库)
func (s *CommandService) Transfer(ctx context.Context, cmd TransferCommand) (string, error) {
	if cmd.FromAccountID == cmd.ToAccountID {
		return "", errors.New("cannot transfer to self")
	}

	// 简单的按ID顺序加锁避免死锁
	firstID, secondID := cmd.FromAccountID, cmd.ToAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	// 这里需要支持事务传递，repository 接口可能需要调整以支持事务，
	// 或者在此层使用 db.Transaction 闭包。
	// 为保持简单，假设 repo 支持 Save 在外部事务中?
	// 目前 repo 接口没有 Transaction 方法。
	// 这里只能先简单调用，如果失败可能导致不一致（生产环境需从 repository 或 unit of work 支持事务）。

	// 由于涉及两个账户，强烈建议将事务控制权上移或在此处使用通过 context 传递的事务
	// 这里演示逻辑：

	return "", errors.New("transfer not implemented yet without transaction support")
}

// saveAccountAndTx 辅助方法：保存账户和流水并发布事件
// 注意：这应该在一个数据库事务中完成，这里简化
func (s *CommandService) saveAccountAndTx(ctx context.Context, account *domain.Account, tx *domain.Transaction) error {
	// 实际应开启事务: tx := db.Begin() -> repo.WithTx(tx).Save(...) -> commit
	// 这里仅做逻辑演示，未保证原子性

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
