package domain

import "context"

// ReferenceDataRepository 参考数据统一仓储接口
type ReferenceDataRepository interface {
	// Symbol
	SaveSymbol(ctx context.Context, symbol *Symbol) error
	GetSymbol(ctx context.Context, id string) (*Symbol, error)
	GetSymbolByCode(ctx context.Context, code string) (*Symbol, error)
	ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*Symbol, error)

	// Exchange
	SaveExchange(ctx context.Context, exchange *Exchange) error
	GetExchange(ctx context.Context, id string) (*Exchange, error)
	ListExchanges(ctx context.Context, limit int, offset int) ([]*Exchange, error)
}
