package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// DepositGatewayRequest 表示充值网关请求参数。
type DepositGatewayRequest struct {
	DepositNo   string
	UserID      string
	AccountID   string
	Amount      decimal.Decimal
	Currency    string
	GatewayType GatewayType
}

// WithdrawalGatewayRequest 表示提现网关请求参数。
type WithdrawalGatewayRequest struct {
	WithdrawalNo  string
	UserID        string
	AccountID     string
	Amount        decimal.Decimal
	NetAmount     decimal.Decimal
	Currency      string
	BankAccountNo string
	BankName      string
	BankHolder    string
}

// FundsGateway 定义账户充值/提现网关接口。
type FundsGateway interface {
	CreateDepositPayment(ctx context.Context, req *DepositGatewayRequest) (string, error)
	Payout(ctx context.Context, req *WithdrawalGatewayRequest) (string, error)
}
