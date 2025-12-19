// Package application 包含参考数据服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/reference-data/domain"
	"github.com/wyfcoding/pkg/logging"
)

// ReferenceDataService 参考数据应用服务
// 负责管理交易对和交易所等基础数据
type ReferenceDataService struct {
	symbolRepo   domain.SymbolRepository   // 交易对仓储接口
	exchangeRepo domain.ExchangeRepository // 交易所仓储接口
}

// NewReferenceDataService 创建应用服务实例
// symbolRepo: 注入的交易对仓储实现
// exchangeRepo: 注入的交易所仓储实现
func NewReferenceDataService(symbolRepo domain.SymbolRepository, exchangeRepo domain.ExchangeRepository) *ReferenceDataService {
	return &ReferenceDataService{
		symbolRepo:   symbolRepo,
		exchangeRepo: exchangeRepo,
	}
}

// GetSymbol 获取交易对
func (s *ReferenceDataService) GetSymbol(ctx context.Context, id string) (*domain.Symbol, error) {
	symbol, err := s.symbolRepo.GetByID(ctx, id)
	if err != nil {
		logging.Error(ctx, "Failed to get symbol",
			"symbol_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}
	return symbol, nil
}

// ListSymbols 列出交易对
func (s *ReferenceDataService) ListSymbols(ctx context.Context, exchangeID string, status string, limit int, offset int) ([]*domain.Symbol, error) {
	symbols, err := s.symbolRepo.List(ctx, exchangeID, status, limit, offset)
	if err != nil {
		logging.Error(ctx, "Failed to list symbols",
			"exchange_id", exchangeID,
			"status", status,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list symbols: %w", err)
	}
	return symbols, nil
}

// GetExchange 获取交易所
func (s *ReferenceDataService) GetExchange(ctx context.Context, id string) (*domain.Exchange, error) {
	exchange, err := s.exchangeRepo.GetByID(ctx, id)
	if err != nil {
		logging.Error(ctx, "Failed to get exchange",
			"exchange_id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get exchange: %w", err)
	}
	return exchange, nil
}

// ListExchanges 列出交易所
func (s *ReferenceDataService) ListExchanges(ctx context.Context, limit int, offset int) ([]*domain.Exchange, error) {
	exchanges, err := s.exchangeRepo.List(ctx, limit, offset)
	if err != nil {
		logging.Error(ctx, "Failed to list exchanges", "error", err)
		return nil, fmt.Errorf("failed to list exchanges: %w", err)
	}
	return exchanges, nil
}
