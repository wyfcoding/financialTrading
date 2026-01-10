// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/goapi/matchingengine/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/pkg/algorithm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler 实现了 MatchingEngineService 的 gRPC 服务端接口。
type Handler struct {
	pb.UnimplementedMatchingEngineServiceServer
	service *application.MatchingEngineService // 关联的撮合应用服务
}

// NewHandler 构造一个新的撮合引擎 gRPC 处理器实例。
func NewHandler(service *application.MatchingEngineService) *Handler {
	return &Handler{
		service: service,
	}
}

// SubmitOrder 处理通过 gRPC 提交的订单请求。
func (h *Handler) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.SubmitOrderResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc submit_order received", "order_id", req.OrderId, "symbol", req.Symbol)

	result, err := h.service.SubmitOrder(ctx, &application.SubmitOrderRequest{
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
	for _, trade := range result.Trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
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
	snapshot, err := h.service.GetOrderBook(ctx, int(req.Depth))
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
	trades, err := h.service.GetTrades(ctx, req.Symbol, int(req.Limit))
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

func (h *Handler) toProtoTrade(trade *algorithm.Trade) *pb.Trade {
	return &pb.Trade{
		TradeId:     trade.TradeID,
		BuyOrderId:  trade.BuyOrderID,
		SellOrderId: trade.SellOrderID,
		Price:       trade.Price.String(),
		Quantity:    trade.Quantity.String(),
		Timestamp:   trade.Timestamp,
	}
}
