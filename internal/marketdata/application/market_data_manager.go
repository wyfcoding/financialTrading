package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataManager 处理所有市场数据相关的写入操作（Commands）。
type MarketDataManager struct {
	quoteRepo     domain.QuoteRepository
	klineRepo     domain.KlineRepository
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
}

// NewMarketDataManager 构造函数。
func NewMarketDataManager(
	quoteRepo domain.QuoteRepository,
	klineRepo domain.KlineRepository,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
) *MarketDataManager {
	return &MarketDataManager{
		quoteRepo:     quoteRepo,
		klineRepo:     klineRepo,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
	}
}

// SaveQuote 保存行情数据
func (m *MarketDataManager) SaveQuote(ctx context.Context, symbol string, bidPrice, askPrice, bidSize, askSize, lastPrice, lastSize decimal.Decimal, timestamp int64, source string) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	quote := &domain.Quote{
		Symbol:    symbol,
		BidPrice:  bidPrice,
		AskPrice:  askPrice,
		BidSize:   bidSize,
		AskSize:   askSize,
		LastPrice: lastPrice,
		LastSize:  lastSize,
		Timestamp: timestamp,
		Source:    source,
	}

	return m.quoteRepo.Save(ctx, quote)
}

// SaveKline 保存 K 线数据
func (m *MarketDataManager) SaveKline(ctx context.Context, kline *domain.Kline) error {
	return m.klineRepo.Save(ctx, kline)
}

// SaveTrade 保存成交记录
func (m *MarketDataManager) SaveTrade(ctx context.Context, trade *domain.Trade) error {
	return m.tradeRepo.Save(ctx, trade)
}

// SaveOrderBook 保存订单簿
func (m *MarketDataManager) SaveOrderBook(ctx context.Context, orderBook *domain.OrderBook) error {
	return m.orderBookRepo.Save(ctx, orderBook)
}
