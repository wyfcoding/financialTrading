// Package application 包含撮合引擎服务的用例逻辑
package application

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialTrading/internal/matching-engine/domain"
	"github.com/wyfcoding/financialTrading/pkg/algos"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// SubmitOrderRequest 提交订单请求 DTO
type SubmitOrderRequest struct {
	OrderID  string // 订单 ID
	Symbol   string // 交易对
	Side     string // 买卖方向
	Price    string // 价格
	Quantity string // 数量
}

// MatchingApplicationService 撮合应用服务
// 负责协调撮合引擎、订单簿和成交记录的持久化
type MatchingApplicationService struct {
	engine        *algos.MatchingEngine      // 撮合引擎核心
	tradeRepo     domain.TradeRepository     // 成交记录仓储接口
	orderBookRepo domain.OrderBookRepository // 订单簿仓储接口
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
		if err := mas.tradeRepo.Save(ctx, trade); err != nil {
			logger.Error(ctx, "Failed to save trade",
				"trade_id", trade.TradeID,
				"error", err,
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

	logger.Debug(ctx, "Order matched successfully",
		"order_id", req.OrderID,
		"trades_count", len(trades),
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
	if err := mas.orderBookRepo.SaveSnapshot(ctx, snapshot); err != nil {
		logger.Error(ctx, "Failed to save order book snapshot",
			"symbol", symbol,
			"error", err,
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

	trades, err := mas.tradeRepo.GetLatest(ctx, symbol, limit)
	if err != nil {
		logger.Error(ctx, "Failed to get trades",
			"symbol", symbol,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	return trades, nil
}
