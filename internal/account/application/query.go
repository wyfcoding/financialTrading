package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/account/domain"
)

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
		AccountID:        a.ID,
		UserID:           a.UserID,
		AccountType:      string(a.AccountType),
		Currency:         a.Currency,
		Balance:          a.Balance.String(),
		AvailableBalance: a.AvailableBalance.String(),
		FrozenBalance:    a.FrozenBalance.String(),
		UpdatedAt:        a.UpdatedAt.Unix(),
		Version:          a.Version,
	}
}
