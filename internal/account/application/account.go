package application

import (
	"context"

	"github.com/shopspring/decimal"
)

// AccountService 作为账户服务操作的门面。
type AccountService struct {
	Command *AccountCommandService
	Query   *AccountQueryService
}

// NewAccountService 创建并返回一个新的 AccountService 门面实例。
func NewAccountService(command *AccountCommandService, query *AccountQueryService) *AccountService {
	return &AccountService{
		Command: command,
		Query:   query,
	}
}

// --- 写操作（委托给 Command） ---

func (s *AccountService) CreateAccount(ctx context.Context, cmd CreateAccountCommand) (*AccountDTO, error) {
	return s.Command.CreateAccount(ctx, cmd)
}

func (s *AccountService) Deposit(ctx context.Context, cmd DepositCommand) error {
	return s.Command.Deposit(ctx, cmd)
}

func (s *AccountService) Freeze(ctx context.Context, cmd FreezeCommand) error {
	return s.Command.Freeze(ctx, cmd)
}

func (s *AccountService) SagaDeductFrozen(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.SagaDeductFrozen(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) SagaRefundFrozen(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.SagaRefundFrozen(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) SagaAddBalance(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.SagaAddBalance(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) SagaSubBalance(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.SagaSubBalance(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) TccTryFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.TccTryFreeze(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) TccConfirmFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.TccConfirmFreeze(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) TccCancelFreeze(ctx context.Context, barrier any, userID, currency string, amount decimal.Decimal) error {
	return s.Command.TccCancelFreeze(ctx, barrier, userID, currency, amount)
}

// --- 读操作（委托给 Query） ---

func (s *AccountService) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	return s.Query.GetAccount(ctx, accountID)
}

func (s *AccountService) GetBalance(ctx context.Context, userID, currency string) (*AccountDTO, error) {
	return s.Query.GetBalance(ctx, userID, currency)
}
