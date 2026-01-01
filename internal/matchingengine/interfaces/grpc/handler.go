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

// GRPCHandler gRPC 处理器
type GRPCHandler struct {
	pb.UnimplementedMatchingEngineServiceServer
	appService *application.MatchingApplicationService
}

func NewGRPCHandler(appService *application.MatchingApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

func (h *GRPCHandler) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.SubmitOrderResponse, error) {
	start := time.Now()
	slog.Info("gRPC SubmitOrder received", "order_id", req.OrderId, "symbol", req.Symbol, "side", req.Side, "price", req.Price, "quantity", req.Quantity)

	result, err := h.appService.SubmitOrder(ctx, &application.SubmitOrderRequest{
		OrderID:  req.OrderId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    req.Price,
		Quantity: req.Quantity,
	})
	if err != nil {
		slog.Error("gRPC SubmitOrder failed", "order_id", req.OrderId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to submit order: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(result.Trades))
	for _, trade := range result.Trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
	}

	slog.Info("gRPC SubmitOrder successful", "order_id", req.OrderId, "trades_count", len(pbTrades), "duration", time.Since(start))
	return &pb.SubmitOrderResponse{
		OrderId:           result.OrderID,
		MatchedTrades:     pbTrades,
		RemainingQuantity: result.RemainingQuantity.String(),
		Status:            result.Status,
	}, nil
}

func (h *GRPCHandler) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.GetOrderBookResponse, error) {
	start := time.Now()
	// 修正：application.GetOrderBook 不再需要 symbol 参数
	snapshot, err := h.appService.GetOrderBook(ctx, int(req.Depth))
	if err != nil {
		slog.Error("gRPC GetOrderBook failed", "error", err, "duration", time.Since(start))
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

	slog.Debug("gRPC GetOrderBook successful", "symbol", snapshot.Symbol, "duration", time.Since(start))
	return &pb.GetOrderBookResponse{
		Symbol:    snapshot.Symbol,
		Bids:      pbBids,
		Asks:      pbAsks,
		Timestamp: snapshot.Timestamp,
	}, nil
}

func (h *GRPCHandler) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.GetTradesResponse, error) {
	start := time.Now()
	trades, err := h.appService.GetTrades(ctx, req.Symbol, int(req.Limit))
	if err != nil {
		slog.Error("gRPC GetTrades failed", "symbol", req.Symbol, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get trades: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(trades))
	for _, trade := range trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
	}

	slog.Debug("gRPC GetTrades successful", "symbol", req.Symbol, "count", len(pbTrades), "duration", time.Since(start))
	return &pb.GetTradesResponse{
		Symbol: req.Symbol,
		Trades: pbTrades,
	}, nil
}

func (h *GRPCHandler) toProtoTrade(trade *algorithm.Trade) *pb.Trade {
	return &pb.Trade{
		TradeId:     trade.TradeID,
		BuyOrderId:  trade.BuyOrderID,
		SellOrderId: trade.SellOrderID,
		Price:       trade.Price.String(),
		Quantity:    trade.Quantity.String(),
		Timestamp:   trade.Timestamp,
	}
}
