package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

// AccountQuery 处理所有账户相关的查询操作（Queries）。
type AccountQuery struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
}

// NewAccountQuery 构造函数。
func NewAccountQuery(accountRepo domain.AccountRepository, transactionRepo domain.TransactionRepository) *AccountQuery {
	return &AccountQuery{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// GetAccount 获取账户信息
func (q *AccountQuery) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	account, err := q.accountRepo.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	return &AccountDTO{
		AccountID:        account.AccountID,
		UserID:           account.UserID,
		AccountType:      account.AccountType,
		Currency:         account.Currency,
		Balance:          account.Balance.String(),
		AvailableBalance: account.AvailableBalance.String(),
		FrozenBalance:    account.FrozenBalance.String(),
		CreatedAt:        account.CreatedAt.Unix(),
		UpdatedAt:        account.UpdatedAt.Unix(),
	}, nil
}

// GetUserAccounts 获取用户的账户列表
func (q *AccountQuery) GetUserAccounts(ctx context.Context, userID string) ([]*AccountDTO, error) {
	accounts, err := q.accountRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*AccountDTO, 0, len(accounts))
	for _, account := range accounts {
		dtos = append(dtos, &AccountDTO{
			AccountID:        account.AccountID,
			UserID:           account.UserID,
			AccountType:      account.AccountType,
			Currency:         account.Currency,
			Balance:          account.Balance.String(),
			AvailableBalance: account.AvailableBalance.String(),
			FrozenBalance:    account.FrozenBalance.String(),
			CreatedAt:        account.CreatedAt.Unix(),
			UpdatedAt:        account.UpdatedAt.Unix(),
		})
	}
	return dtos, nil
}

// GetTransactionHistory 获取交易历史
func (q *AccountQuery) GetTransactionHistory(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	return q.transactionRepo.GetHistory(ctx, accountID, limit, offset)
}
