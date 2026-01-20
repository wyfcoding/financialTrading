package application

import (
	"github.com/shopspring/decimal"
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

// AccountDTO 账户信息传输对象
type AccountDTO struct {
	AccountID        string
	UserID           string
	AccountType      string
	Currency         string
	Balance          string
	AvailableBalance string
	FrozenBalance    string
	UpdatedAt        int64
	Version          int64
}

// TransactionDTO 流水传输对象
type TransactionDTO struct {
	TransactionID string
	AccountID     string
	Type          string
	Amount        string
	Status        string
	Timestamp     int64
}
