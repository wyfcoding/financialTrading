package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

// AccountService 账户门面服务，整合 Manager 和 Query。
type AccountService struct {
	manager *AccountManager
	query   *AccountQuery
}

// NewAccountService 构造函数。
func NewAccountService(accountRepo domain.AccountRepository, transactionRepo domain.TransactionRepository) *AccountService {
	return &AccountService{
		manager: NewAccountManager(accountRepo, transactionRepo),
		query:   NewAccountQuery(accountRepo, transactionRepo),
	}
}

// --- Manager (Writes) ---

func (s *AccountService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*AccountDTO, error) {
	return s.manager.CreateAccount(ctx, req)
}

func (s *AccountService) Deposit(ctx context.Context, accountID string, amount decimal.Decimal) error {
	return s.manager.Deposit(ctx, accountID, amount)
}

func (s *AccountService) FreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal, reason string) error {
	return s.manager.FreezeBalance(ctx, accountID, amount, reason)
}

func (s *AccountService) UnfreezeBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	return s.manager.UnfreezeBalance(ctx, accountID, amount)
}

func (s *AccountService) DeductFrozenBalance(ctx context.Context, accountID string, amount decimal.Decimal) error {
	return s.manager.DeductFrozenBalance(ctx, accountID, amount)
}

// --- TCC Facade ---

func (s *AccountService) TccTryFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return s.manager.TccTryFreeze(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) TccConfirmFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return s.manager.TccConfirmFreeze(ctx, barrier, userID, currency, amount)
}

func (s *AccountService) TccCancelFreeze(ctx context.Context, barrier interface{}, userID, currency string, amount decimal.Decimal) error {
	return s.manager.TccCancelFreeze(ctx, barrier, userID, currency, amount)
}

// --- Query (Reads) ---

func (s *AccountService) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	return s.query.GetAccount(ctx, accountID)
}

func (s *AccountService) GetUserAccounts(ctx context.Context, userID string) ([]*AccountDTO, error) {
	return s.query.GetUserAccounts(ctx, userID)
}

func (s *AccountService) GetTransactionHistory(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	return s.query.GetTransactionHistory(ctx, accountID, limit, offset)
}

// --- Legacy Compatibility Types ---

// CreateAccountRequest 创建账户请求 DTO
type CreateAccountRequest struct {
	UserID      string // 用户 ID
	AccountType string // 账户类型
	Currency    string // 币种
}

// AccountDTO 账户 DTO
type AccountDTO struct {
	AccountID        string
	UserID           string
	AccountType      string
	Currency         string
	Balance          string
	AvailableBalance string
	FrozenBalance    string
	CreatedAt        int64
	UpdatedAt        int64
}
