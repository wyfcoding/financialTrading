package application

import (
	"context"
	"fmt"
	"log/slog"

	clearingv1 "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algorithm"
)

// MatchingEngineService 撮合门面服务，整合 Manager 和 Query。
type MatchingEngineService struct {
	manager *MatchingEngineManager
	query   *MatchingEngineQuery
}

// NewMatchingEngineService 构造函数。
func NewMatchingEngineService(symbol string, tradeRepo domain.TradeRepository, orderBookRepo domain.OrderBookRepository, logger *slog.Logger) (*MatchingEngineService, error) {
	// 初始化 Disruptor 模式引擎
	engine, err := domain.NewDisruptionEngine(symbol, 1048576, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init matching engine: %w", err)
	}

	return &MatchingEngineService{
		manager: NewMatchingEngineManager(symbol, engine, tradeRepo, orderBookRepo, logger),
		query:   NewMatchingEngineQuery(engine, tradeRepo),
	}, nil
}

func (s *MatchingEngineService) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	s.manager.SetClearingClient(cli)
}

// --- Manager (Writes) ---

func (s *MatchingEngineService) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	return s.manager.SubmitOrder(ctx, req)
}

// --- Query (Reads) ---

func (s *MatchingEngineService) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	return s.query.GetOrderBook(ctx, depth)
}

func (s *MatchingEngineService) GetTrades(ctx context.Context, symbol string, limit int) ([]*algorithm.Trade, error) {
	return s.query.GetTrades(ctx, symbol, limit)
}

// --- Legacy Compatibility Types ---

// SubmitOrderRequest 提交订单请求 DTO
type SubmitOrderRequest struct {
	OrderID  string // 订单 ID
	Symbol   string // 交易对
	Side     string // 买卖方向
	Price    string // 价格
	Quantity string // 数量
}
