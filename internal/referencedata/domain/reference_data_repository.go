package domain

import (
	"context"
)

// SymbolRepository 交易对仓储接口
type SymbolRepository interface {
	Save(ctx context.Context, symbol *Symbol) error
	GetByID(ctx context.Context, id string) (*Symbol, error)
	GetByCode(ctx context.Context, code string) (*Symbol, error)
	List(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*Symbol, error)
}

// ExchangeRepository 交易所仓储接口
type ExchangeRepository interface {
	Save(ctx context.Context, exchange *Exchange) error
	GetByID(ctx context.Context, id string) (*Exchange, error)
	List(ctx context.Context, limit int, offset int) ([]*Exchange, error)
}
