package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/matchingengine/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler 实现了 gRPC 服务端接口。
type Handler struct {
	pb.UnimplementedMatchingEngineServiceServer
	cmd   *application.MatchingCommandService
	query *application.MatchingQueryService
}

// NewHandler 构造新的 gRPC 处理器实例。
func NewHandler(cmd *application.MatchingCommandService, query *application.MatchingQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

// SubmitOrder 处理通过 gRPC 提交的订单请求。
func (h *Handler) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.SubmitOrderResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc submit_order received", "order_id", req.OrderId, "symbol", req.Symbol)

	result, err := h.cmd.SubmitOrder(ctx, &application.SubmitOrderCommand{
		OrderID:                req.OrderId,
		Symbol:                 req.Symbol,
		Side:                   req.Side,
		Price:                  req.Price,
		Quantity:               req.Quantity,
		UserID:                 req.UserId,
		IsIceberg:              req.IsIceberg,
		IcebergDisplayQuantity: req.IcebergDisplayQuantity,
		PostOnly:               req.PostOnly,
	})
	if err != nil {
		slog.ErrorContext(ctx, "grpc submit_order failed", "order_id", req.OrderId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to submit order: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(result.Trades))
	for _, t := range result.Trades {
		domainTrade := &domain.Trade{
			TradeID:     t.TradeID,
			BuyOrderID:  t.BuyOrderID,
			SellOrderID: t.SellOrderID,
			Symbol:      t.Symbol,
			Price:       t.Price.InexactFloat64(),
			Quantity:    t.Quantity.InexactFloat64(),
			Timestamp:   time.Unix(0, t.Timestamp),
		}
		pbTrades = append(pbTrades, h.toProtoTrade(domainTrade))
	}

	slog.InfoContext(ctx, "grpc submit_order successful", "order_id", req.OrderId, "trades_count", len(pbTrades), "duration", time.Since(start))
	return &pb.SubmitOrderResponse{
		OrderId:           result.OrderID,
		MatchedTrades:     pbTrades,
		RemainingQuantity: result.RemainingQuantity.String(),
		Status:            result.Status,
	}, nil
}

// GetOrderBook 返回当前内存订单簿的聚合深度快照。
func (h *Handler) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.GetOrderBookResponse, error) {
	start := time.Now()
	snapshot, err := h.query.GetOrderBook(ctx, int(req.Depth))
	if err != nil {
		slog.ErrorContext(ctx, "grpc get_order_book failed", "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get order book: %v", err)
	}

	pbBids := make([]*pb.OrderBookLevel, 0, len(snapshot.Bids))
	for _, bid := range snapshot.Bids {
		pbBids = append(pbBids, &pb.OrderBookLevel{
			Price:    bid.Price.String(),
			Quantity: bid.Quantity.String(),
		})
	}

	pbAsks := make([]*pb.OrderBookLevel, 0, len(snapshot.Asks))
	for _, ask := range snapshot.Asks {
		pbAsks = append(pbAsks, &pb.OrderBookLevel{
			Price:    ask.Price.String(),
			Quantity: ask.Quantity.String(),
		})
	}

	slog.DebugContext(ctx, "grpc get_order_book successful", "symbol", snapshot.Symbol, "duration", time.Since(start))
	return &pb.GetOrderBookResponse{
		Symbol:    snapshot.Symbol,
		Bids:      pbBids,
		Asks:      pbAsks,
		Timestamp: snapshot.Timestamp,
	}, nil
}

// GetTrades 获取指定交易对的最近成交历史记录。
func (h *Handler) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.GetTradesResponse, error) {
	start := time.Now()
	trades, err := h.query.GetTrades(ctx, req.Symbol, int(req.Limit))
	if err != nil {
		slog.ErrorContext(ctx, "grpc get_trades failed", "symbol", req.Symbol, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get trades: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(trades))
	for _, trade := range trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
	}

	slog.DebugContext(ctx, "grpc get_trades successful", "symbol", req.Symbol, "count", len(pbTrades), "duration", time.Since(start))
	return &pb.GetTradesResponse{
		Symbol: req.Symbol,
		Trades: pbTrades,
	}, nil
}

func (h *Handler) toProtoTrade(trade *domain.Trade) *pb.Trade {
	return &pb.Trade{
		TradeId:     trade.TradeID,
		BuyOrderId:  trade.BuyOrderID,
		SellOrderId: trade.SellOrderID,
		Price:       fmt.Sprintf("%f", trade.Price),
		Quantity:    fmt.Sprintf("%f", trade.Quantity),
		Timestamp:   trade.Timestamp.UnixNano(),
	}
}
