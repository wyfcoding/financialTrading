package mysql

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"gorm.io/gorm"
)

// AccountPO 账户持久化对象 (Snapshot)
type AccountPO struct {
	gorm.Model
	AccountID        string          `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null"`
	AccountType      string          `gorm:"column:account_type;type:varchar(20);not null"`
	Currency         string          `gorm:"column:currency;type:varchar(10);not null"`
	Balance          decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null"`
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null"`
	FrozenBalance    decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null"`
	Version          int64           `gorm:"column:version;default:0;not null"`
}

func (AccountPO) TableName() string {
	return "accounts"
}

// ToDomain 转换为领域对象
func (po *AccountPO) ToDomain() *domain.Account {
	return &domain.Account{
		ID:               po.AccountID,
		UserID:           po.UserID,
		AccountType:      domain.AccountType(po.AccountType),
		Currency:         po.Currency,
		Balance:          po.Balance,
		AvailableBalance: po.AvailableBalance,
		FrozenBalance:    po.FrozenBalance,
		Version:          po.Version,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}

// FromDomain 从领域对象转换
func (po *AccountPO) FromDomain(a *domain.Account) {
	po.AccountID = a.ID
	po.UserID = a.UserID
	po.AccountType = string(a.AccountType)
	po.Currency = a.Currency
	po.Balance = a.Balance
	po.AvailableBalance = a.AvailableBalance
	po.FrozenBalance = a.FrozenBalance
	po.Version = a.Version
}

// TransactionPO 交易流水 (Read Model)
type TransactionPO struct {
	gorm.Model
	TransactionID string          `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null"`
	AccountID     string          `gorm:"column:account_id;type:varchar(32);index;not null"`
	Type          string          `gorm:"column:type;type:varchar(20);not null"`
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null"`
	Status        string          `gorm:"column:status;type:varchar(20);not null"`
	Timestamp     int64           `gorm:"column:timestamp;not null"`
}

func (TransactionPO) TableName() string {
	return "transactions"
}

// EventPO 事件存储对象
type EventPO struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	AggregateID string `gorm:"column:aggregate_id;type:varchar(32);index;not null"`
	EventType   string `gorm:"column:event_type;type:varchar(50);not null"`
	Payload     string `gorm:"column:payload;type:json;not null"`
	OccurredAt  int64  `gorm:"column:occurred_at;not null"`
}

func (EventPO) TableName() string {
	return "account_events"
}
