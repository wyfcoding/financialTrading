// Package application 包含撮合引擎的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/fynnwu/FinancialTrading/internal/matching-engine/domain"
	"github.com/fynnwu/FinancialTrading/pkg/algos"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// SubmitOrderRequest 提交订单请求 DTO
type SubmitOrderRequest struct {
	OrderID  string
	Symbol   string
	Side     string
	Price    string
	Quantity string
}

// MatchingApplicationService 撮合应用服务
type MatchingApplicationService struct {
	engine        *algos.MatchingEngine
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
}

// NewMatchingApplicationService 创建撮合应用服务
func NewMatchingApplicationService(tradeRepo domain.TradeRepository, orderBookRepo domain.OrderBookRepository) *MatchingApplicationService {
	return &MatchingApplicationService{
		engine:        algos.NewMatchingEngine(),
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
	}
}

// SubmitOrder 提交订单进行撮合
// 用例流程：
// 1. 验证订单参数
// 2. 创建订单对象
// 3. 执行撮合算法
// 4. 保存成交记录
// 5. 返回撮合结果
func (mas *MatchingApplicationService) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	// 验证输入
	if req.OrderID == "" || req.Symbol == "" || req.Side == "" {
		return nil, fmt.Errorf("invalid request parameters")
	}

	// 解析价格和数量
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 创建订单对象
	order := &algos.Order{
		OrderID:   req.OrderID,
		Symbol:    req.Symbol,
		Side:      req.Side,
		Price:     price,
		Quantity:  quantity,
		Timestamp: 0, // 实际应用中应使用当前时间戳
	}

	// 执行撮合
	trades := mas.engine.Match(order)

	// 保存成交记录
	for _, trade := range trades {
		if err := mas.tradeRepo.Save(trade); err != nil {
			logger.WithContext(ctx).Error("Failed to save trade",
				zap.String("trade_id", trade.TradeID),
				zap.Error(err),
			)
		}
	}

	// 构建撮合结果
	result := &domain.MatchingResult{
		OrderID:           req.OrderID,
		Trades:            trades,
		RemainingQuantity: order.Quantity,
		Status:            "MATCHED",
	}

	logger.WithContext(ctx).Debug("Order matched successfully",
		zap.String("order_id", req.OrderID),
		zap.Int("trades_count", len(trades)),
	)

	return result, nil
}

// GetOrderBook 获取订单簿快照
func (mas *MatchingApplicationService) GetOrderBook(ctx context.Context, symbol string, depth int) (*domain.OrderBookSnapshot, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	if depth <= 0 {
		depth = 20
	}

	// 从引擎获取订单簿
	bids := mas.engine.GetBids(depth)
	asks := mas.engine.GetAsks(depth)

	// 转换为快照
	snapshot := &domain.OrderBookSnapshot{
		Symbol:    symbol,
		Bids:      make([]*domain.OrderBookLevel, 0, len(bids)),
		Asks:      make([]*domain.OrderBookLevel, 0, len(asks)),
		Timestamp: 0, // 实际应用中应使用当前时间戳
	}

	for _, bid := range bids {
		snapshot.Bids = append(snapshot.Bids, &domain.OrderBookLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		})
	}

	for _, ask := range asks {
		snapshot.Asks = append(snapshot.Asks, &domain.OrderBookLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		})
	}

	// 保存快照
	if err := mas.orderBookRepo.SaveSnapshot(snapshot); err != nil {
		logger.WithContext(ctx).Error("Failed to save order book snapshot",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
	}

	return snapshot, nil
}

// GetTrades 获取成交历史
func (mas *MatchingApplicationService) GetTrades(ctx context.Context, symbol string, limit int) ([]*algos.Trade, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	if limit <= 0 {
		limit = 100
	}

	trades, err := mas.tradeRepo.GetLatest(symbol, limit)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get trades",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	return trades, nil
}
