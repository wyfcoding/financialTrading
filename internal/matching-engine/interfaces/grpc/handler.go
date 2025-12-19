// Package grpc 包含 gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/matching-engine"
	"github.com/wyfcoding/financialTrading/internal/matching-engine/application"
	"github.com/wyfcoding/pkg/algorithm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与撮合引擎相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedMatchingEngineServiceServer
	appService *application.MatchingApplicationService // 撮合应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的撮合应用服务
func NewGRPCHandler(appService *application.MatchingApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// SubmitOrder 提交订单
// 处理 gRPC SubmitOrder 请求
func (h *GRPCHandler) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.MatchResult, error) {
	// 调用应用服务提交订单
	result, err := h.appService.SubmitOrder(ctx, &application.SubmitOrderRequest{
		OrderID:  req.OrderId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    req.Price,
		Quantity: req.Quantity,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to submit order: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(result.Trades))
	for _, trade := range result.Trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
	}

	return &pb.MatchResult{
		OrderId:           result.OrderID,
		MatchedTrades:     pbTrades,
		RemainingQuantity: result.RemainingQuantity.String(),
		Status:            result.Status,
	}, nil
}

// GetOrderBook 获取订单簿
// 处理 gRPC GetOrderBook 请求
func (h *GRPCHandler) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.OrderBookSnapshot, error) {
	snapshot, err := h.appService.GetOrderBook(ctx, req.Symbol, int(req.Depth))
	if err != nil {
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

	return &pb.OrderBookSnapshot{
		Symbol:    snapshot.Symbol,
		Bids:      pbBids,
		Asks:      pbAsks,
		Timestamp: snapshot.Timestamp,
	}, nil
}

// GetTrades 获取成交记录
// 处理 gRPC GetTrades 请求
func (h *GRPCHandler) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.TradesResponse, error) {
	trades, err := h.appService.GetTrades(ctx, req.Symbol, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get trades: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(trades))
	for _, trade := range trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
	}

	return &pb.TradesResponse{
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
