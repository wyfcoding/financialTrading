package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

// AccountDTO 账户信息传输对象
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

// AccountQueryService 处理账户相关的读操作。
type AccountQueryService struct {
	repo domain.AccountRepository
}

func NewAccountQueryService(repo domain.AccountRepository) *AccountQueryService {
	return &AccountQueryService{repo: repo}
}

func (q *AccountQueryService) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	account, err := q.repo.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}
	return q.toDTO(account), nil
}

func (q *AccountQueryService) GetBalance(ctx context.Context, userID, currency string) (*AccountDTO, error) {
	// 简单的查询逻辑，实际可能需要专门的 Read Model 优化
	accounts, err := q.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, acc := range accounts {
		if acc.Currency == currency {
			return q.toDTO(acc), nil
		}
	}
	return nil, fmt.Errorf("account not found")
}

func (q *AccountQueryService) toDTO(a *domain.Account) *AccountDTO {
	return &AccountDTO{
		AccountID:        a.AccountID,
		UserID:           a.UserID,
		AccountType:      string(a.AccountType),
		Currency:         a.Currency,
		Balance:          a.Balance.String(),
		AvailableBalance: a.AvailableBalance.String(),
		FrozenBalance:    a.FrozenBalance.String(),
		CreatedAt:        a.CreatedAt.Unix(),
		UpdatedAt:        a.UpdatedAt.Unix(),
		Version:          a.Version(),
	}
}
