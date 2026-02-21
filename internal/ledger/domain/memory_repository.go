package domain

import (
	"context"
	"errors"
	"sync"
	"time"
)

var errJournalNotFound = errors.New("journal not found")

type MemoryLedgerRepository struct {
	mu         sync.RWMutex
	accounts   map[string]*Account
	journalsTx map[string]*Journal
	holds      map[string]*FundHold
}

func NewMemoryLedgerRepository() *MemoryLedgerRepository {
	return &MemoryLedgerRepository{
		accounts:   make(map[string]*Account),
		journalsTx: make(map[string]*Journal),
		holds:      make(map[string]*FundHold),
	}
}

func (r *MemoryLedgerRepository) GetAccount(_ context.Context, id string) (*Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	acc, ok := r.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	cp := *acc
	return &cp, nil
}

func (r *MemoryLedgerRepository) GetAccountForUpdate(ctx context.Context, id string) (*Account, error) {
	return r.GetAccount(ctx, id)
}

func (r *MemoryLedgerRepository) UpdateBalance(_ context.Context, acc *Account) error {
	if acc == nil || acc.ID == "" {
		return ErrAccountNotFound
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *acc
	r.accounts[acc.ID] = &cp
	return nil
}

func (r *MemoryLedgerRepository) SaveJournal(_ context.Context, j *Journal) error {
	if j == nil || j.TransactionID == "" {
		return errors.New("invalid journal")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *j
	cp.Entries = append([]LedgerEntry(nil), j.Entries...)
	r.journalsTx[j.TransactionID] = &cp
	return nil
}

func (r *MemoryLedgerRepository) GetJournalByTransaction(_ context.Context, txID string) (*Journal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	j, ok := r.journalsTx[txID]
	if !ok {
		return nil, errJournalNotFound
	}
	cp := *j
	cp.Entries = append([]LedgerEntry(nil), j.Entries...)
	return &cp, nil
}

func (r *MemoryLedgerRepository) CreateHold(_ context.Context, hold *FundHold) error {
	if hold == nil || hold.HoldID == "" {
		return errors.New("invalid hold")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *hold
	r.holds[hold.HoldID] = &cp
	return nil
}

func (r *MemoryLedgerRepository) GetHoldForUpdate(_ context.Context, holdID string) (*FundHold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hold, ok := r.holds[holdID]
	if !ok {
		return nil, ErrHoldNotFound
	}
	cp := *hold
	return &cp, nil
}

func (r *MemoryLedgerRepository) UpdateHold(_ context.Context, hold *FundHold) error {
	if hold == nil || hold.HoldID == "" {
		return ErrHoldNotFound
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.holds[hold.HoldID]; !ok {
		return ErrHoldNotFound
	}

	cp := *hold
	r.holds[hold.HoldID] = &cp
	return nil
}

func (r *MemoryLedgerRepository) GenerateEODSnapshot(_ context.Context, _ time.Time) error {
	return nil
}

var _ LedgerRepository = (*MemoryLedgerRepository)(nil)
