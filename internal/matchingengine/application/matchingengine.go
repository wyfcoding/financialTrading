package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wyfcoding/pkg/algorithm/types"

	clearingv1 "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

// MatchingEngineService 撮合门面服务，整合 Manager 和 Query。
type MatchingEngineService struct {
	manager *MatchingEngineManager
	query   *MatchingEngineQuery
	logger  *slog.Logger
}

// NewMatchingEngineService 构造函数。
func NewMatchingEngineService(
	symbol string,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
	db *gorm.DB,
	outboxMgr *outbox.Manager,
	logger *slog.Logger,
) (*MatchingEngineService, error) {
	// 初始化 Disruptor 模式引擎
	engine, err := domain.NewDisruptionEngine(symbol, 1048576, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init matching engine: %w", err)
	}

	return &MatchingEngineService{
		manager: NewMatchingEngineManager(symbol, engine, tradeRepo, orderBookRepo, db, outboxMgr, logger),
		query:   NewMatchingEngineQuery(engine, tradeRepo),
		logger:  logger,
	}, nil
}

func (s *MatchingEngineService) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	s.manager.SetClearingClient(cli)
}

func (s *MatchingEngineService) SetOrderClient(cli orderv1.OrderServiceClient) {
	s.manager.SetOrderClient(cli)
	// 在设置 Client 后立即尝试恢复状态 (或者在 main.go 中手动触发)
	ctx := context.Background()
	if err := s.manager.RecoverState(ctx); err != nil {
		s.logger.Error("failed to recover engine state", "error", err)
	}
}

// --- Manager (Writes) ---

func (s *MatchingEngineService) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	return s.manager.SubmitOrder(ctx, req)
}

// --- Query (Reads) ---

func (s *MatchingEngineService) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	return s.query.GetOrderBook(ctx, depth)
}

func (s *MatchingEngineService) GetTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error) {
	return s.query.GetTrades(ctx, symbol, limit)
}

// --- Legacy Compatibility Types ---

// SubmitOrderRequest 提交订单请求 DTO
type SubmitOrderRequest struct {
	OrderID                string // 订单 ID
	Symbol                 string // 交易对
	Side                   string // 买卖方向
	Price                  string // 价格
	Quantity               string // 数量
	UserID                 string // 所有人 ID
	IsIceberg              bool   // 是否为冰山单
	IcebergDisplayQuantity string // 冰山单显性规模
	PostOnly               bool   // 是否为只做 Maker
}
