package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/eventsourcing"
	"gorm.io/gorm"
)

// accountRepository 账户仓储实现
type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository 创建并返回一个新的 accountRepository 实例。
func NewAccountRepository(db *gorm.DB) domain.AccountRepository {
	return &accountRepository{db: db}
}

// --- tx helpers ---

func (r *accountRepository) BeginTx(ctx context.Context) any {
	return r.db.WithContext(ctx).Begin()
}

func (r *accountRepository) CommitTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Commit().Error
}

func (r *accountRepository) RollbackTx(tx any) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok || gormTx == nil {
		return errors.New("invalid transaction")
	}
	return gormTx.Rollback().Error
}

func (r *accountRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

// Save 保存账户（带乐观锁）
func (r *accountRepository) Save(ctx context.Context, account *domain.Account) error {
	db := r.getDB(ctx)

	// 新建账户
	if account.ID == 0 {
		model := toAccountModel(account)
		if err := db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		account.ID = model.ID
		account.CreatedAt = model.CreatedAt
		account.UpdatedAt = model.UpdatedAt
		return nil
	}

	currentVersion := account.Version()
	result := db.WithContext(ctx).Model(&AccountModel{}).
		Where("account_id = ? AND version = ?", account.AccountID, currentVersion).
		Updates(map[string]any{
			"balance":           account.Balance,
			"available_balance": account.AvailableBalance,
			"frozen_balance":    account.FrozenBalance,
			"borrowed_amount":   account.BorrowedAmount,
			"locked_collateral": account.LockedCollateral,
			"accrued_interest":  account.AccruedInterest,
			"version":           currentVersion + 1,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock failed: account modified by another transaction")
	}

	account.SetVersion(currentVersion + 1)
	account.UpdatedAt = time.Now()
	return nil
}

func (r *accountRepository) Get(ctx context.Context, id string) (*domain.Account, error) {
	var model AccountModel
	if err := r.getDB(ctx).WithContext(ctx).Where("account_id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toAccount(&model), nil
}

func (r *accountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Account, error) {
	var models []*AccountModel
	if err := r.getDB(ctx).WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	accounts := make([]*domain.Account, len(models))
	for i, m := range models {
		accounts[i] = toAccount(m)
	}
	return accounts, nil
}

func (r *accountRepository) List(ctx context.Context, accType domain.AccountType, limit, offset int) ([]*domain.Account, error) {
	var models []*AccountModel
	query := r.getDB(ctx).WithContext(ctx)
	if accType != "" {
		query = query.Where("account_type = ?", accType)
	}
	if err := query.Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}
	accounts := make([]*domain.Account, len(models))
	for i, m := range models {
		accounts[i] = toAccount(m)
	}
	return accounts, nil
}

func (r *accountRepository) ExecWithBarrier(ctx context.Context, barrier any, fn func(ctx context.Context) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		return fn(txCtx)
	})
}

func (r *accountRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// --- 事件存储实现 ---

type eventStore struct {
	db *gorm.DB
}

func NewEventStore(db *gorm.DB) domain.EventStore {
	return &eventStore{db: db}
}

func (s *eventStore) Save(ctx context.Context, aggregateID string, events []eventsourcing.DomainEvent, expectedVersion int64) error {
	db := s.getDB(ctx)

	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		po := &EventPO{
			AggregateID: aggregateID,
			EventType:   event.EventType(),
			Payload:     string(payload),
			OccurredAt:  event.OccurredAt().UnixNano(),
		}

		if err := db.Create(po).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *eventStore) Load(ctx context.Context, aggregateID string) ([]eventsourcing.DomainEvent, error) {
	return nil, nil
}

func (s *eventStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextx.GetTx(ctx).(*gorm.DB); ok {
		return tx
	}
	return s.db
}
