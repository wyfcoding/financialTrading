package mysql

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
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

func (r *accountRepository) Save(ctx context.Context, account *domain.Account) error {
	db := r.getDB(ctx)

	// 使用 gorm.Model 的 ID 判断新老对象
	if account.Model.ID == 0 {
		return db.Create(account).Error
	}

	// 乐观锁更新 (使用 AggregateRoot.Version())
	currentVersion := account.Version()
	result := db.Model(&domain.Account{}).
		Where("account_id = ? AND version = ?", account.AccountID, currentVersion).
		Updates(map[string]interface{}{
			"balance":           account.Balance,
			"available_balance": account.AvailableBalance,
			"frozen_balance":    account.FrozenBalance,
			"version":           currentVersion + 1,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock failed: account modified by another transaction")
	}

	account.SetVersion(currentVersion + 1)
	return nil
}

func (r *accountRepository) Get(ctx context.Context, id string) (*domain.Account, error) {
	var acc domain.Account
	if err := r.getDB(ctx).Where("account_id = ?", id).First(&acc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	// 设置聚合 ID
	acc.SetID(acc.AccountID)
	return &acc, nil
}

func (r *accountRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Account, error) {
	var accounts []*domain.Account
	if err := r.getDB(ctx).Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	for _, acc := range accounts {
		acc.SetID(acc.AccountID)
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

// --- 持久化对象 (Infrastructure POs) ---

// TransactionPO 交易流水
type TransactionPO struct {
	gorm.Model
	TransactionID string          `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null;comment:交易ID"`
	AccountID     string          `gorm:"column:account_id;type:varchar(32);index;not null;comment:账户ID"`
	Type          string          `gorm:"column:type;type:varchar(20);not null;comment:类型"`
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null;comment:金额"`
	Status        string          `gorm:"column:status;type:varchar(20);not null;comment:状态"`
	Timestamp     int64           `gorm:"column:timestamp;not null;comment:时间戳"`
}

func (TransactionPO) TableName() string {
	return "transactions"
}

// EventPO 事件存储对象
type EventPO struct {
	gorm.Model
	AggregateID string `gorm:"column:aggregate_id;type:varchar(32);index;not null;comment:聚合ID"`
	EventType   string `gorm:"column:event_type;type:varchar(50);not null;comment:事件类型"`
	Payload     string `gorm:"column:payload;type:json;not null;comment:事件负载"`
	OccurredAt  int64  `gorm:"column:occurred_at;not null;comment:发生时间"`
}

func (EventPO) TableName() string {
	return "account_events"
}
