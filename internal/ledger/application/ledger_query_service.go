package application

import (
	"context"
	"time"

	"github.com/wyfcoding/financialtrading/internal/ledger/domain"
)

type LedgerQueryService struct {
	repo       domain.LedgerRepository
	readRepo   domain.LedgerReadRepository
	searchRepo domain.LedgerSearchRepository
}

func NewLedgerQueryService(repo domain.LedgerRepository, r, s, l interface{}) *LedgerQueryService {
	_ = l
	svc := &LedgerQueryService{repo: repo}
	if rr, ok := r.(domain.LedgerReadRepository); ok {
		svc.readRepo = rr
	}
	if sr, ok := s.(domain.LedgerSearchRepository); ok {
		svc.searchRepo = sr
	}
	return svc
}

func (s *LedgerQueryService) GetBalance(ctx context.Context, id, cur string) (*domain.Account, error) {
	return s.repo.GetAccount(ctx, id)
}

func (s *LedgerQueryService) GetAccountSummary(ctx context.Context, ownerID string) (any, error) {
	if s.readRepo != nil {
		return s.readRepo.GetAccountSummary(ctx, ownerID)
	}
	type accountSummaryGetter interface {
		ListAccountsByOwner(ctx context.Context, ownerID string) ([]*domain.Account, error)
	}
	if getter, ok := s.repo.(accountSummaryGetter); ok {
		return getter.ListAccountsByOwner(ctx, ownerID)
	}
	return []*domain.Account{}, nil
}

func (s *LedgerQueryService) GetStatement(ctx context.Context, id, cur string, start, end time.Time, page, pageSize int) (*domain.AccountStatement, error) {
	type statementGetter interface {
		GetStatement(ctx context.Context, id, cur string, s, e time.Time, l, o int) (*domain.AccountStatement, error)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if getter, ok := s.repo.(statementGetter); ok {
		return getter.GetStatement(ctx, id, cur, start, end, pageSize, offset)
	}
	return &domain.AccountStatement{
		AccountID: id,
		Currency:  cur,
		StartDate: start,
		EndDate:   end,
		Page:      page,
		PageSize:  pageSize,
		Entries:   []domain.LedgerEntry{},
	}, nil
}

func (s *LedgerQueryService) GetActiveHolds(ctx context.Context, accountID string) ([]*domain.FundHold, error) {
	type activeHoldLister interface {
		ListActiveHolds(ctx context.Context, id string) ([]*domain.FundHold, error)
	}
	if getter, ok := s.repo.(activeHoldLister); ok {
		return getter.ListActiveHolds(ctx, accountID)
	}
	return []*domain.FundHold{}, nil
}

func (s *LedgerQueryService) GetJournal(ctx context.Context, journalID string) (*domain.Journal, error) {
	type journalGetter interface {
		GetJournal(ctx context.Context, id string) (*domain.Journal, error)
	}
	if getter, ok := s.repo.(journalGetter); ok {
		return getter.GetJournal(ctx, journalID)
	}
	return s.repo.GetJournalByTransaction(ctx, journalID)
}

func (s *LedgerQueryService) GetTrialBalance(ctx context.Context, date time.Time) (*domain.TrialBalance, error) {
	type trialBalanceGetter interface {
		GetTrialBalance(ctx context.Context, d time.Time) (*domain.TrialBalance, error)
	}
	if getter, ok := s.repo.(trialBalanceGetter); ok {
		return getter.GetTrialBalance(ctx, date)
	}
	return &domain.TrialBalance{
		AsOf:     date,
		Items:    []domain.TrialBalanceItem{},
		Balanced: true,
	}, nil
}

func (s *LedgerQueryService) GetAuditTrail(ctx context.Context, accountID string, start, end time.Time) ([]*domain.AuditTrailItem, error) {
	if s.searchRepo != nil {
		return s.searchRepo.GetAuditTrail(ctx, accountID, start, end)
	}
	return []*domain.AuditTrailItem{}, nil
}

func (s *LedgerQueryService) SearchEntries(ctx context.Context, query *domain.EntrySearchQuery) (*domain.EntrySearchResult, error) {
	if query == nil {
		query = &domain.EntrySearchQuery{}
	}
	if s.searchRepo != nil {
		return s.searchRepo.SearchEntries(ctx, query)
	}
	return &domain.EntrySearchResult{
		Entries:  []*domain.EntrySearchDoc{},
		Total:    0,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *LedgerQueryService) SearchJournals(ctx context.Context, query *domain.JournalSearchQuery) (*domain.JournalSearchResult, error) {
	if query == nil {
		query = &domain.JournalSearchQuery{}
	}
	if s.searchRepo != nil {
		return s.searchRepo.SearchJournals(ctx, query)
	}
	return &domain.JournalSearchResult{
		Journals: []*domain.JournalSearchDoc{},
		Total:    0,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}
