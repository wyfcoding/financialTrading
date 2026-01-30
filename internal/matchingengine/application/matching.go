package application

import (
	"context"
	"fmt"
	"log/slog"

	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"gorm.io/gorm"
)

// MatchingService 作为撮合引擎操作的门面。
type MatchingService struct {
	Command *MatchingCommandService
	Query   *MatchingQueryService
	logger  *slog.Logger
}

// NewMatchingService 构造函数。
func NewMatchingService(
	symbol string,
	tradeRepo domain.TradeRepository,
	orderBookRepo domain.OrderBookRepository,
	db *gorm.DB,
	publisher domain.EventPublisher,
	logger *slog.Logger,
) (*MatchingService, error) {
	// 初始化 Disruptor 模式引擎
	engine, err := domain.NewDisruptionEngine(symbol, 1048576, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init matching engine: %w", err)
	}

	return &MatchingService{
		Command: NewMatchingCommandService(symbol, engine, tradeRepo, orderBookRepo, db, publisher, logger),
		Query:   NewMatchingQueryService(engine, tradeRepo),
		logger:  logger,
	}, nil
}

func (s *MatchingService) SetClearingClient(cli clearingv1.ClearingServiceClient) {
	s.Command.SetClearingClient(cli)
}

func (s *MatchingService) SetOrderClient(cli orderv1.OrderServiceClient) {
	s.Command.SetOrderClient(cli)
	// 在设置 Client 后立即尝试恢复状态
	ctx := context.Background()
	if err := s.Command.RecoverState(ctx); err != nil {
		s.logger.Error("failed to recover engine state", "error", err)
	}
}

// --- 写操作 (Delegates to Command) ---

func (s *MatchingService) SubmitOrder(ctx context.Context, req *SubmitOrderRequest) (*domain.MatchingResult, error) {
	return s.Command.SubmitOrder(ctx, req)
}

// --- 读操作 (Delegates to Query) ---

func (s *MatchingService) GetOrderBook(ctx context.Context, depth int) (*domain.OrderBookSnapshot, error) {
	return s.Query.GetOrderBook(ctx, depth)
}

func (s *MatchingService) GetTrades(ctx context.Context, symbol string, limit int) ([]*domain.Trade, error) {
	return s.Query.GetTrades(ctx, symbol, limit)
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
