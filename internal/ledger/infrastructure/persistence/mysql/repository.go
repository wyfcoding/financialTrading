package mysql

import (
	"context"
	"time"
	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
	"github.com/wyfcoding/pkg/database"
)

type LedgerRepository struct {
	db *database.DB
}

func NewLedgerRepository(db *database.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) CreateAccount(ctx context.Context, acc *domain.Account) error { return nil }
func (r *LedgerRepository) GetAccount(ctx context.Context, id string) (*domain.Account, error) { return nil, nil }
func (r *LedgerRepository) GetAccountForUpdate(ctx context.Context, id string) (*domain.Account, error) { return nil, nil }
func (r *LedgerRepository) UpdateBalance(ctx context.Context, acc *domain.Account) error { return nil }
func (r *LedgerRepository) ListAccountsByOwner(ctx context.Context, id string) ([]*domain.Account, error) { return nil, nil }
func (r *LedgerRepository) SaveJournal(ctx context.Context, j *domain.Journal) error { return nil }
func (r *LedgerRepository) GetJournal(ctx context.Context, id string) (*domain.Journal, error) { return nil, nil }
func (r *LedgerRepository) GetJournalByTransaction(ctx context.Context, id string) (*domain.Journal, error) { return nil, nil }
func (r *LedgerRepository) MarkJournalReversed(ctx context.Context, id string) error { return nil }
func (r *LedgerRepository) SaveHold(ctx context.Context, h *domain.FundHold) error { return nil }
func (r *LedgerRepository) GetHoldByReference(ctx context.Context, ref string) (*domain.FundHold, error) { return nil, nil }
func (r *LedgerRepository) UpdateHold(ctx context.Context, h *domain.FundHold) error { return nil }
func (r *LedgerRepository) ListActiveHolds(ctx context.Context, id string) ([]*domain.FundHold, error) { return nil, nil }
func (r *LedgerRepository) GetStatement(ctx context.Context, id, cur string, s, e time.Time, l, o int) (*domain.AccountStatement, error) { return nil, nil }
func (r *LedgerRepository) GetTrialBalance(ctx context.Context, d time.Time) (*domain.TrialBalance, error) { return nil, nil }
