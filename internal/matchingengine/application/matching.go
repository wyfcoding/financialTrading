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

// --- DTO Definitions ---

// SubmitOrderCommand 提交订单命令 DTO
type SubmitOrderCommand struct {
	OrderID                string `json:"order_id"`
	Symbol                 string `json:"symbol"`
	Side                   string `json:"side"` // "buy" or "sell"
	Price                  string `json:"price"`
	Quantity               string `json:"quantity"`
	UserID                 string `json:"user_id"`
	IsIceberg              bool   `json:"is_iceberg"`
	IcebergDisplayQuantity string `json:"iceberg_display_quantity"`
	PostOnly               bool   `json:"post_only"`
}

type OrderBookDTO struct {
	Symbol    string     `json:"symbol"`
	Bids      []LevelDTO `json:"bids"`
	Asks      []LevelDTO `json:"asks"`
	Timestamp int64      `json:"timestamp"`
}

type LevelDTO struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

type TradeDTO struct {
	TradeID      string `json:"trade_id"`
	MakerOrderID string `json:"maker_order_id"`
	TakerOrderID string `json:"taker_order_id"`
	Symbol       string `json:"symbol"`
	Price        string `json:"price"`
	Quantity     string `json:"quantity"`
	Timestamp    int64  `json:"timestamp"`
}

// SubmitOrder 提交订单
func (s *MatchingService) SubmitOrder(ctx context.Context, cmd *SubmitOrderCommand) (*domain.MatchingResult, error) {
	return s.Command.SubmitOrder(ctx, cmd)
}
