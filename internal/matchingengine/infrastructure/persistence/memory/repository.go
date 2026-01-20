package memory

import (
	"context"
	"sync"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
)

type inMemoryRepository struct {
	snapshots map[string]*domain.OrderBookSnapshot
	mu        sync.Mutex
}

func NewInMemoryRepository() domain.OrderBookRepository {
	return &inMemoryRepository{
		snapshots: make(map[string]*domain.OrderBookSnapshot),
	}
}

func (r *inMemoryRepository) SaveSnapshot(ctx context.Context, snapshot *domain.OrderBookSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots[snapshot.Symbol] = snapshot
	return nil
}

func (r *inMemoryRepository) LoadSnapshot(ctx context.Context, symbol string) (*domain.OrderBookSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if snap, ok := r.snapshots[symbol]; ok {
		return snap, nil
	}
	return nil, nil // Return nil if not found
}
