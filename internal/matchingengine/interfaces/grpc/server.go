package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/matchingengine/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedMatchingEngineServiceServer
	app *application.MatchingEngineManager
}

func NewServer(s *grpc.Server, app *application.MatchingEngineManager) *Server {
	srv := &Server{app: app}
	v1.RegisterMatchingEngineServiceServer(s, srv)
	return srv
}

func (s *Server) SubmitOrder(ctx context.Context, req *v1.SubmitOrderRequest) (*v1.SubmitOrderResponse, error) {
	// app expecting *SubmitOrderRequest with string price/qty as per Step 2426 analysis for MatchingEngineManager
	// Using the updated struct definition I assumed:
	// But Step 2426 showed: price, err := decimal.NewFromString(req.Price) in application logic.
	// So app takes struct with Price string.

	cmd := &application.SubmitOrderRequest{
		OrderID:                req.OrderId,
		Symbol:                 req.Symbol,
		Side:                   req.Side,
		Price:                  req.Price,
		Quantity:               req.Quantity,
		UserID:                 req.UserId,
		IsIceberg:              req.IsIceberg,
		IcebergDisplayQuantity: req.IcebergDisplayQuantity,
		PostOnly:               req.PostOnly,
	}

	result, err := s.app.SubmitOrder(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "submit order failed: %v", err)
	}

	trades := make([]*v1.Trade, len(result.Trades))
	for i, t := range result.Trades {
		trades[i] = &v1.Trade{
			TradeId:     t.TradeID,
			BuyOrderId:  t.BuyOrderID,
			SellOrderId: t.SellOrderID,
			Price:       t.Price.String(),
			Quantity:    t.Quantity.String(),
			Timestamp:   t.Timestamp,
		}
	}

	return &v1.SubmitOrderResponse{
		OrderId:           req.OrderId,
		MatchedTrades:     trades,
		RemainingQuantity: result.RemainingQuantity.String(),
		Status:            string(result.Status),
	}, nil
}

func (s *Server) GetOrderBook(ctx context.Context, req *v1.GetOrderBookRequest) (*v1.GetOrderBookResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetOrderBook not implemented")
}

func (s *Server) GetTrades(ctx context.Context, req *v1.GetTradesRequest) (*v1.GetTradesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetTrades not implemented")
}

func (s *Server) SubscribeTrades(req *v1.SubscribeTradesRequest, stream v1.MatchingEngineService_SubscribeTradesServer) error {
	return status.Error(codes.Unimplemented, "method SubscribeTrades not implemented")
}
