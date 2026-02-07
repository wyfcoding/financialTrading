package domain

import "context"

// ReferenceDataRepository 参考数据统一仓储接口（写模型）
type ReferenceDataRepository interface {
	BeginTx(ctx context.Context) any
	CommitTx(tx any) error
	RollbackTx(tx any) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// Symbol
	SaveSymbol(ctx context.Context, symbol *Symbol) error
	DeleteSymbol(ctx context.Context, id string) error
	GetSymbol(ctx context.Context, id string) (*Symbol, error)
	GetSymbolByCode(ctx context.Context, code string) (*Symbol, error)
	ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*Symbol, error)

	// Exchange
	SaveExchange(ctx context.Context, exchange *Exchange) error
	DeleteExchange(ctx context.Context, id string) error
	GetExchange(ctx context.Context, id string) (*Exchange, error)
	GetExchangeByName(ctx context.Context, name string) (*Exchange, error)
	ListExchanges(ctx context.Context, limit int, offset int) ([]*Exchange, error)

	// Instrument
	SaveInstrument(ctx context.Context, instrument *Instrument) error
	DeleteInstrument(ctx context.Context, symbol string) error
	GetInstrument(ctx context.Context, symbol string) (*Instrument, error)
	ListInstruments(ctx context.Context, limit int, offset int) ([]*Instrument, error)
}

// SymbolReadRepository 交易对读模型缓存
type SymbolReadRepository interface {
	Save(ctx context.Context, symbol *Symbol) error
	Get(ctx context.Context, id string) (*Symbol, error)
	GetByCode(ctx context.Context, code string) (*Symbol, error)
}

// ExchangeReadRepository 交易所读模型缓存
type ExchangeReadRepository interface {
	Save(ctx context.Context, exchange *Exchange) error
	Get(ctx context.Context, id string) (*Exchange, error)
	GetByName(ctx context.Context, name string) (*Exchange, error)
}

// InstrumentReadRepository 合约读模型缓存
type InstrumentReadRepository interface {
	Save(ctx context.Context, instrument *Instrument) error
	Get(ctx context.Context, symbol string) (*Instrument, error)
}

// ReferenceDataSearchRepository 提供基于 Elasticsearch 的参考数据搜索能力
type ReferenceDataSearchRepository interface {
	IndexSymbol(ctx context.Context, symbol *Symbol) error
	IndexExchange(ctx context.Context, exchange *Exchange) error
	IndexInstrument(ctx context.Context, instrument *Instrument) error
	SearchSymbols(ctx context.Context, exchangeID, status, keyword string, limit, offset int) ([]*Symbol, int64, error)
	SearchExchanges(ctx context.Context, name, country, status string, limit, offset int) ([]*Exchange, int64, error)
	SearchInstruments(ctx context.Context, symbol, instrumentType string, limit, offset int) ([]*Instrument, int64, error)
}
