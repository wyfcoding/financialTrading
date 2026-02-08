// Package application 资金服务查询服务
package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
)

// QueryService 资金查询服务
type QueryService struct {
	accountRepo domain.AccountRepository
	txRepo      domain.TransactionRepository
	logger      *slog.Logger
}

// NewQueryService 创建查询服务
func NewQueryService(
	accountRepo domain.AccountRepository,
	txRepo domain.TransactionRepository,
	logger *slog.Logger,
) *QueryService {
	return &QueryService{
		accountRepo: accountRepo,
		txRepo:      txRepo,
		logger:      logger,
	}
}

// AccountDTO 账户 DTO
type AccountDTO struct {
	AccountID uint64
	OwnerID   uint64
	Type      domain.AccountType
	Currency  domain.Currency
	Balance   int64
	Available int64
	Frozen    int64
	Status    domain.AccountStatus
	UpdatedAt time.Time
}

// TransactionDTO 流水 DTO
type TransactionDTO struct {
	TransactionID string
	AccountID     uint
	Type          domain.TransactionType
	Amount        int64
	BalanceAfter  int64
	ReferenceID   string
	Remark        string
	CreatedAt     time.Time
}

// GetBalance 获取余额
func (s *QueryService) GetBalance(ctx context.Context, accountID uint64) (*AccountDTO, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.toAccountDTO(account), nil
}

// ListTransactions 获取流水
func (s *QueryService) ListTransactions(ctx context.Context, accountID uint64, txType *domain.TransactionType, page, pageSize int) ([]TransactionDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	txs, total, err := s.txRepo.List(ctx, uint(accountID), txType, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]TransactionDTO, 0, len(txs))
	for _, tx := range txs {
		dtos = append(dtos, *s.toTransactionDTO(tx))
	}

	return dtos, total, nil
}

func (s *QueryService) toAccountDTO(acc *domain.Account) *AccountDTO {
	return &AccountDTO{
		AccountID: uint64(acc.ID),
		OwnerID:   acc.OwnerID,
		Type:      acc.Type,
		Currency:  acc.Currency,
		Balance:   acc.Balance,
		Available: acc.Available,
		Frozen:    acc.Frozen,
		Status:    acc.Status,
		UpdatedAt: acc.UpdatedAt,
	}
}

func (s *QueryService) toTransactionDTO(tx *domain.Transaction) *TransactionDTO {
	return &TransactionDTO{
		TransactionID: tx.TransactionID,
		AccountID:     tx.AccountID,
		Type:          tx.Type,
		Amount:        tx.Amount,
		BalanceAfter:  tx.BalanceAfter,
		ReferenceID:   tx.ReferenceID,
		Remark:        tx.Remark,
		CreatedAt:     tx.CreatedAt,
	}
}
