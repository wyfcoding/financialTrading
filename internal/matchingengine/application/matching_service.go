package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm"
	"github.com/wyfcoding/pkg/logging"
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
	engine        *domain.DisruptionEngine
	tradeRepo     domain.TradeRepository
	orderBookRepo domain.OrderBookRepository
	logger        *slog.Logger
}

// NewMatchingApplicationService 创建撮合应用服务
func NewMatchingApplicationService(symbol string, tradeRepo domain.TradeRepository, orderBookRepo domain.OrderBookRepository, logger *slog.Logger) (*MatchingApplicationService, error) {
	// 初始化 Disruptor 模式引擎，容量设置为 1024*1024
	engine, err := domain.NewDisruptionEngine(symbol, 1048576, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init matching engine: %w", err)
	}

	return &MatchingApplicationService{
		engine:        engine,
		tradeRepo:     tradeRepo,
		orderBookRepo: orderBookRepo,
		logger:        logger.With("module", "matching_engine", "symbol", symbol),
	}, nil
}

// SubmitOrder 提交订单进行撮合
func (mas *MatchingApplicationService) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	// 记录性能监控
	defer logging.LogDuration(ctx, "Order matching processing finished",
		"order_id", req.OrderID,
		"symbol", req.Symbol,
	)()

	// 验证与解析
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	// 封装为算法层订单对象
	order := &algorithm.Order{
		OrderID:   req.OrderID,
		Symbol:    req.Symbol,
		Side:      req.Side,
		Price:     price,
		Quantity:  quantity,
		Timestamp: time.Now().UnixNano(),
	}

	// 提交至 Disruptor 引擎
	mas.logger.Debug("submitting order to disruption engine", "order_id", order.OrderID, "side", order.Side, "price", order.Price.String(), "qty", order.Quantity.String())
	result, err := mas.engine.SubmitOrder(order)
	if err != nil {
		mas.logger.Error("failed to submit order to engine", "order_id", order.OrderID, "error", err)
		return nil, err
	}

	mas.logger.Info("order processed by engine", "order_id", order.OrderID, "status", result.Status, "trades_count", len(result.Trades), "remaining_qty", result.RemainingQuantity.String())

	// 异步持久化成交记录，不阻塞核心撮合路径
	if len(result.Trades) > 0 {
		mas.asyncPersistTrades(result.Trades)
	}

	return result, nil
}

// asyncPersistTrades 异步持久化成交记录
func (mas *MatchingApplicationService) asyncPersistTrades(trades []*algorithm.Trade) {
	mas.logger.Debug("starting async persistence of trades", "count", len(trades))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, t := range trades {
			if err := mas.tradeRepo.Save(ctx, t); err != nil {
				mas.logger.Error("failed to persist trade", "trade_id", t.TradeID, "error", err)
			} else {
				mas.logger.Debug("trade persisted successfully", "trade_id", t.TradeID)
			}
		}
	}()
}

// GetOrderBook 获取订单簿快照
func (mas *MatchingApplicationService) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	if depth <= 0 {
		depth = 20
	}

	snapshot := mas.engine.GetOrderBookSnapshot(depth)

	// 保存快照
	if err := mas.orderBookRepo.SaveSnapshot(ctx, snapshot); err != nil {
		mas.logger.Error("failed to save order book snapshot", "error", err)
	}

	return snapshot, nil
}

// GetTrades 获取成交历史
func (mas *MatchingApplicationService) GetTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	if limit <= 0 {
		limit = 100
	}

	return mas.tradeRepo.GetLatestTrades(ctx, symbol, limit)
}
