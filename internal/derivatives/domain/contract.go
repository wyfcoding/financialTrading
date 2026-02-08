package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ContractType string
type ContractStatus int8

const (
	TypeCall   ContractType = "CALL"
	TypePut    ContractType = "PUT"
	TypeFuture ContractType = "FUTURE"

	StatusTrading ContractStatus = 1
	StatusSettled ContractStatus = 2
	StatusExpired ContractStatus = 3
)

func (s ContractStatus) String() string {
	switch s {
	case StatusTrading:
		return "TRADING"
	case StatusSettled:
		return "SETTLED"
	case StatusExpired:
		return "EXPIRED"
	}
	return "UNKNOWN"
}

// Contract 衍生品合约
type Contract struct {
	gorm.Model
	ContractID  string          `gorm:"column:contract_id;type:varchar(32);unique_index;not null"`
	Symbol      string          `gorm:"column:symbol;type:varchar(32);unique_index;not null"` // e.g. BTC-240329-30000-C
	Underlying  string          `gorm:"column:underlying;type:varchar(10);index;not null"`    // BTC
	Type        ContractType    `gorm:"column:type;type:varchar(10);not null"`
	StrikePrice decimal.Decimal `gorm:"column:strike_price;type:decimal(20,8);not null"`
	ExpiryDate  time.Time       `gorm:"column:expiry_date;index;not null"`
	Multiplier  decimal.Decimal `gorm:"column:multiplier;type:decimal(10,4);not null"`
	Status      ContractStatus  `gorm:"column:status;type:tinyint;not null;default:1"`
}

func (Contract) TableName() string { return "contracts" }

func NewContract(id, symbol, underlying string, cType ContractType, strike decimal.Decimal, expiry time.Time, mult decimal.Decimal) *Contract {
	return &Contract{
		ContractID:  id,
		Symbol:      symbol,
		Underlying:  underlying,
		Type:        cType,
		StrikePrice: strike,
		ExpiryDate:  expiry,
		Multiplier:  mult,
		Status:      StatusTrading,
	}
}

func (c *Contract) IsExpired() bool {
	return time.Now().After(c.ExpiryDate)
}

func (c *Contract) Settle() error {
	if c.Status != StatusTrading {
		return errors.New("contract not in trading status")
	}
	c.Status = StatusSettled
	return nil
}

func (c *Contract) Expire() {
	if c.Status == StatusTrading {
		c.Status = StatusExpired
	}
}
