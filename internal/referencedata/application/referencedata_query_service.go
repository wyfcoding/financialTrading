package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

// ReferenceDataQueryService 处理所有参考数据相关的查询操作（Queries）。
type ReferenceDataQueryService struct {
	repo               domain.ReferenceDataRepository
	symbolReadRepo     domain.SymbolReadRepository
	exchangeReadRepo   domain.ExchangeReadRepository
	instrumentReadRepo domain.InstrumentReadRepository
	searchRepo         domain.ReferenceDataSearchRepository
}

// NewReferenceDataQueryService 构造函数。
func NewReferenceDataQueryService(
	repo domain.ReferenceDataRepository,
	symbolReadRepo domain.SymbolReadRepository,
	exchangeReadRepo domain.ExchangeReadRepository,
	instrumentReadRepo domain.InstrumentReadRepository,
	searchRepo domain.ReferenceDataSearchRepository,
) *ReferenceDataQueryService {
	return &ReferenceDataQueryService{
		repo:               repo,
		symbolReadRepo:     symbolReadRepo,
		exchangeReadRepo:   exchangeReadRepo,
		instrumentReadRepo: instrumentReadRepo,
		searchRepo:         searchRepo,
	}
}

// GetSymbol 获取单个交易对
func (s *ReferenceDataQueryService) GetSymbol(ctx context.Context, idOrCode string) (*SymbolDTO, error) {
	if idOrCode == "" {
		return nil, nil
	}
	if s.symbolReadRepo != nil {
		if cached, err := s.symbolReadRepo.Get(ctx, idOrCode); err == nil && cached != nil {
			return toSymbolDTO(cached), nil
		}
		if cached, err := s.symbolReadRepo.GetByCode(ctx, idOrCode); err == nil && cached != nil {
			return toSymbolDTO(cached), nil
		}
	}

	symbol, err := s.repo.GetSymbol(ctx, idOrCode)
	if err != nil || symbol == nil {
		if err != nil {
			return nil, err
		}
		symbol, err = s.repo.GetSymbolByCode(ctx, idOrCode)
		if err != nil {
			return nil, err
		}
	}
	if symbol != nil && s.symbolReadRepo != nil {
		_ = s.symbolReadRepo.Save(ctx, symbol)
	}
	return toSymbolDTO(symbol), nil
}

// ListSymbols 列表查询
func (s *ReferenceDataQueryService) ListSymbols(ctx context.Context, exchangeID, status string, limit int, offset int) ([]*SymbolDTO, error) {
	symbols, err := s.repo.ListSymbols(ctx, exchangeID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	return toSymbolDTOs(symbols), nil
}

// GetExchange 获取交易所信息
func (s *ReferenceDataQueryService) GetExchange(ctx context.Context, idOrName string) (*ExchangeDTO, error) {
	if idOrName == "" {
		return nil, nil
	}
	if s.exchangeReadRepo != nil {
		if cached, err := s.exchangeReadRepo.Get(ctx, idOrName); err == nil && cached != nil {
			return toExchangeDTO(cached), nil
		}
		if cached, err := s.exchangeReadRepo.GetByName(ctx, idOrName); err == nil && cached != nil {
			return toExchangeDTO(cached), nil
		}
	}

	exchange, err := s.repo.GetExchange(ctx, idOrName)
	if err != nil || exchange == nil {
		if err != nil {
			return nil, err
		}
		exchange, err = s.repo.GetExchangeByName(ctx, idOrName)
		if err != nil {
			return nil, err
		}
	}
	if exchange != nil && s.exchangeReadRepo != nil {
		_ = s.exchangeReadRepo.Save(ctx, exchange)
	}
	return toExchangeDTO(exchange), nil
}

// ListExchanges 交易所列表
func (s *ReferenceDataQueryService) ListExchanges(ctx context.Context, limit int, offset int) ([]*ExchangeDTO, error) {
	exchanges, err := s.repo.ListExchanges(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return toExchangeDTOs(exchanges), nil
}

// GetInstrument 获取合约
func (s *ReferenceDataQueryService) GetInstrument(ctx context.Context, symbol string) (*InstrumentDTO, error) {
	if symbol == "" {
		return nil, nil
	}
	if s.instrumentReadRepo != nil {
		if cached, err := s.instrumentReadRepo.Get(ctx, symbol); err == nil && cached != nil {
			return toInstrumentDTO(cached), nil
		}
	}
	instrument, err := s.repo.GetInstrument(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if instrument != nil && s.instrumentReadRepo != nil {
		_ = s.instrumentReadRepo.Save(ctx, instrument)
	}
	return toInstrumentDTO(instrument), nil
}

// ListInstruments 列出合约
func (s *ReferenceDataQueryService) ListInstruments(ctx context.Context, limit int, offset int) ([]*InstrumentDTO, error) {
	instruments, err := s.repo.ListInstruments(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return toInstrumentDTOs(instruments), nil
}

// SearchSymbols 交易对搜索
func (s *ReferenceDataQueryService) SearchSymbols(ctx context.Context, exchangeID, status, keyword string, limit, offset int) ([]*SymbolDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	symbols, total, err := s.searchRepo.SearchSymbols(ctx, exchangeID, status, keyword, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toSymbolDTOs(symbols), total, nil
}

// SearchExchanges 交易所搜索
func (s *ReferenceDataQueryService) SearchExchanges(ctx context.Context, name, country, status string, limit, offset int) ([]*ExchangeDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	exchanges, total, err := s.searchRepo.SearchExchanges(ctx, name, country, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toExchangeDTOs(exchanges), total, nil
}

// SearchInstruments 合约搜索
func (s *ReferenceDataQueryService) SearchInstruments(ctx context.Context, symbol, instrumentType string, limit, offset int) ([]*InstrumentDTO, int64, error) {
	if s.searchRepo == nil {
		return nil, 0, nil
	}
	instruments, total, err := s.searchRepo.SearchInstruments(ctx, symbol, instrumentType, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toInstrumentDTOs(instruments), total, nil
}
