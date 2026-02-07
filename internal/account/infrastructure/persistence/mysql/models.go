package mysql

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
	"gorm.io/gorm"
)

// AccountModel 账户写模型。
type AccountModel struct {
	gorm.Model
	AccountID        string          `gorm:"column:account_id;type:varchar(32);uniqueIndex;not null;comment:账户ID"`
	UserID           string          `gorm:"column:user_id;type:varchar(32);index;not null;comment:用户ID"`
	AccountType      string          `gorm:"column:account_type;type:varchar(20);not null;comment:账户类型"`
	Currency         string          `gorm:"column:currency;type:varchar(10);not null;comment:币种"`
	Balance          decimal.Decimal `gorm:"column:balance;type:decimal(32,18);default:0;not null;comment:总余额"`
	AvailableBalance decimal.Decimal `gorm:"column:available_balance;type:decimal(32,18);default:0;not null;comment:可用余额"`
	FrozenBalance    decimal.Decimal `gorm:"column:frozen_balance;type:decimal(32,18);default:0;not null;comment:冻结余额"`
	Version          int64           `gorm:"column:version;not null;default:0;comment:聚合版本"`
}

func (AccountModel) TableName() string { return "accounts" }

// TransactionPO 交易流水
// 保留写模型，便于后续扩展。
type TransactionPO struct {
	gorm.Model
	TransactionID string          `gorm:"column:transaction_id;type:varchar(32);uniqueIndex;not null;comment:交易ID"`
	AccountID     string          `gorm:"column:account_id;type:varchar(32);index;not null;comment:账户ID"`
	Type          string          `gorm:"column:type;type:varchar(20);not null;comment:类型"`
	Amount        decimal.Decimal `gorm:"column:amount;type:decimal(32,18);not null;comment:金额"`
	Status        string          `gorm:"column:status;type:varchar(20);not null;comment:状态"`
	Timestamp     int64           `gorm:"column:timestamp;not null;comment:时间戳"`
}

func (TransactionPO) TableName() string { return "transactions" }

// EventPO 事件存储对象
type EventPO struct {
	gorm.Model
	AggregateID string `gorm:"column:aggregate_id;type:varchar(32);index;not null;comment:聚合ID"`
	EventType   string `gorm:"column:event_type;type:varchar(50);not null;comment:事件类型"`
	Payload     string `gorm:"column:payload;type:json;not null;comment:事件负载"`
	OccurredAt  int64  `gorm:"column:occurred_at;not null;comment:发生时间"`
}

func (EventPO) TableName() string { return "account_events" }

func toAccountModel(account *domain.Account) *AccountModel {
	if account == nil {
		return nil
	}
	return &AccountModel{
		Model: gorm.Model{
			ID:        account.ID,
			CreatedAt: account.CreatedAt,
			UpdatedAt: account.UpdatedAt,
		},
		AccountID:        account.AccountID,
		UserID:           account.UserID,
		AccountType:      string(account.AccountType),
		Currency:         account.Currency,
		Balance:          account.Balance,
		AvailableBalance: account.AvailableBalance,
		FrozenBalance:    account.FrozenBalance,
		Version:          account.Version(),
	}
}

func toAccount(model *AccountModel) *domain.Account {
	if model == nil {
		return nil
	}
	acc := &domain.Account{
		ID:               model.ID,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
		AccountID:        model.AccountID,
		UserID:           model.UserID,
		AccountType:      domain.AccountType(model.AccountType),
		Currency:         model.Currency,
		Balance:          model.Balance,
		AvailableBalance: model.AvailableBalance,
		FrozenBalance:    model.FrozenBalance,
	}
	acc.SetID(acc.AccountID)
	acc.SetVersion(model.Version)
	return acc
}
