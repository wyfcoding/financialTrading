// 包  gRPC 处理器实现
package grpc

import (
	"context"

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

	return &pb.SubmitOrderResponse{
		OrderId:           result.OrderID,
		MatchedTrades:     pbTrades,
		RemainingQuantity: result.RemainingQuantity.String(),
		Status:            result.Status,
	}, nil
}

func (h *GRPCHandler) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.GetOrderBookResponse, error) {
	// 修正：application.GetOrderBook 不再需要 symbol 参数
	snapshot, err := h.appService.GetOrderBook(ctx, int(req.Depth))
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

	return &pb.GetOrderBookResponse{
		Symbol:    snapshot.Symbol,
		Bids:      pbBids,
		Asks:      pbAsks,
		Timestamp: snapshot.Timestamp,
	}, nil
}

func (h *GRPCHandler) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.GetTradesResponse, error) {
	trades, err := h.appService.GetTrades(ctx, req.Symbol, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get trades: %v", err)
	}

	pbTrades := make([]*pb.Trade, 0, len(trades))
	for _, trade := range trades {
		pbTrades = append(pbTrades, h.toProtoTrade(trade))
	}

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
