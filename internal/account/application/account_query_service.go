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
	BorrowedAmount   string
	LockedCollateral string
	AccruedInterest  string
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
	repo     domain.AccountRepository
	readRepo domain.AccountReadRepository
}

func NewAccountQueryService(repo domain.AccountRepository, readRepo domain.AccountReadRepository) *AccountQueryService {
	return &AccountQueryService{repo: repo, readRepo: readRepo}
}

func (q *AccountQueryService) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	if q.readRepo != nil {
		if cached, err := q.readRepo.Get(ctx, accountID); err == nil && cached != nil {
			return q.toDTO(cached), nil
		}
	}
	account, err := q.repo.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}
	if q.readRepo != nil {
		_ = q.readRepo.Save(ctx, account)
	}
	return q.toDTO(account), nil
}

func (q *AccountQueryService) GetBalance(ctx context.Context, userID, currency string) (*AccountDTO, error) {
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

func (q *AccountQueryService) ListAccounts(ctx context.Context, accType string, pageSize, pageToken int) ([]*AccountDTO, int, error) {
	accounts, err := q.repo.List(ctx, domain.AccountType(accType), pageSize, pageToken)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*AccountDTO, len(accounts))
	for i, acc := range accounts {
		dtos[i] = q.toDTO(acc)
	}

	nextPageToken := 0
	if len(accounts) == pageSize {
		nextPageToken = pageToken + pageSize
	}

	return dtos, nextPageToken, nil
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
		BorrowedAmount:   a.BorrowedAmount.String(),
		LockedCollateral: a.LockedCollateral.String(),
		AccruedInterest:  a.AccruedInterest.String(),
		CreatedAt:        a.CreatedAt.Unix(),
		UpdatedAt:        a.UpdatedAt.Unix(),
		Version:          a.Version(),
	}
}
