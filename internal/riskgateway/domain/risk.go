package domain

import (
	"sync"

	"github.com/wyfcoding/pkg/money"
)

type AccountRisk struct {
	AccountID    string
	MaxOrderSize money.Money
	MaxDailyLoss money.Money
}

func (r *AccountRisk) CheckOrder(orderAmount money.Money) bool {
	if r == nil {
		return false
	}
	return orderAmount.ToFen() <= r.MaxOrderSize.ToFen()
}

type RiskRepository interface {
	GetAccountRisk(accountID string) (*AccountRisk, bool)
	SaveAccountRisk(risk *AccountRisk)
}

type MemoryRiskRepository struct {
	mu    sync.RWMutex
	risks map[string]*AccountRisk
}

func NewMemoryRiskRepository() *MemoryRiskRepository {
	return &MemoryRiskRepository{
		risks: make(map[string]*AccountRisk),
	}
}

func (r *MemoryRiskRepository) GetAccountRisk(accountID string) (*AccountRisk, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	risk, ok := r.risks[accountID]
	if !ok {
		return nil, false
	}
	return risk, true
}

func (r *MemoryRiskRepository) SaveAccountRisk(risk *AccountRisk) {
	if risk == nil || risk.AccountID == "" {
		return
	}
	r.mu.Lock()
	r.risks[risk.AccountID] = risk
	r.mu.Unlock()
}

var _ RiskRepository = (*MemoryRiskRepository)(nil)
