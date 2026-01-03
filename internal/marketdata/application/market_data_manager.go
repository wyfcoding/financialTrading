package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

// MarketDataManager 处理所有市场数据相关的写入操作（Commands）。
type MarketDataManager struct {
	quoteRepo     domain.QuoteRepository
	klineRepo     domain.KlineRepository
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	logger        *slog.Logger
}

// NewMarketDataManager 构造函数。
func NewMarketDataManager(
	quoteRepo domain.QuoteRepository,
	klineRepo domain.KlineRepository,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
	logger *slog.Logger,
) *MarketDataManager {
	return &MarketDataManager{
		quoteRepo:     quoteRepo,
		klineRepo:     klineRepo,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		logger:        logger.With("module", "market_data_manager"),
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

// HandleTradeExecuted 处理成交事件并增量更新 K 线和行情记录
func (m *MarketDataManager) HandleTradeExecuted(ctx context.Context, event map[string]any) error {
	tradeID := event["trade_id"].(string)
	symbol := event["symbol"].(string)
	price, _ := decimal.NewFromString(event["price"].(string))
	quantity, _ := decimal.NewFromString(event["quantity"].(string))
	executedAt := event["executed_at"].(int64) // 假设为纳秒

	m.logger.Debug("processing trade for market data", "trade_id", tradeID, "symbol", symbol)

	// 1. 保存成交快照 (用于行情服务的成交历史拉取)
	trade := &domain.Trade{
		TradeID:   tradeID,
		Symbol:    symbol,
		Price:     price,
		Quantity:  quantity,
		Timestamp: executedAt / 1e6, // 统一转为毫秒
		Source:    "MATCHING_ENGINE",
	}
	if err := m.tradeRepo.Save(ctx, trade); err != nil {
		return fmt.Errorf("failed to save trade snapshot: %w", err)
	}

	// 2. 增量更新 K 线 (此处实现 1m K 线增量逻辑)
	return m.updateKline(ctx, symbol, price, quantity, executedAt)
}

func (m *MarketDataManager) updateKline(ctx context.Context, symbol string, price, quantity decimal.Decimal, timestamp int64) error {
	// 将时间戳对齐到分钟 (假设 timestamp 是纳秒)
	intervalSec := int64(60)
	openTime := (timestamp / 1e9 / intervalSec) * intervalSec * 1000 // 毫秒
	closeTime := openTime + (intervalSec * 1000) - 1

	// 尝试获取当前分钟的 K 线
	klines, err := m.klineRepo.Get(ctx, symbol, "1m", openTime, openTime+1000)
	if err != nil {
		return err
	}

	var kline *domain.Kline
	tradeValue := price.Mul(quantity)

	if len(klines) == 0 {
		// 创建新 K 线
		kline = &domain.Kline{
			Symbol:           symbol,
			Interval:         "1m",
			OpenTime:         openTime,
			CloseTime:        closeTime,
			Open:             price,
			High:             price,
			Low:              price,
			Close:            price,
			Volume:           quantity,
			QuoteAssetVolume: tradeValue,
			TradeCount:       1,
		}
	} else {
		// 增量更新现有 K 线
		kline = klines[0]
		kline.Close = price
		kline.Volume = kline.Volume.Add(quantity)
		kline.QuoteAssetVolume = kline.QuoteAssetVolume.Add(tradeValue)
		kline.TradeCount++
		
		if price.GreaterThan(kline.High) {
			kline.High = price
		}
		if price.LessThan(kline.Low) {
			kline.Low = price
		}
	}

	return m.klineRepo.Save(ctx, kline)
}
