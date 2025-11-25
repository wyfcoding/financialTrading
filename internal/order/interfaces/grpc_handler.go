package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/order"
	"github.com/wyfcoding/financialTrading/internal/order/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedOrderServiceServer
	appService *application.OrderApplicationService
}

func NewGRPCHandler(appService *application.OrderApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

func (h *GRPCHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	dto, err := h.appService.CreateOrder(ctx, &application.CreateOrderRequest{
		UserID:        req.UserId,
		Symbol:        req.Symbol,
		Side:          req.Side,
		OrderType:     req.OrderType,
		Price:         req.Price,
		Quantity:      req.Quantity,
		TimeInForce:   req.TimeInForce,
		ClientOrderID: req.ClientOrderId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	return h.toProtoResponse(dto), nil
}

func (h *GRPCHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.OrderResponse, error) {
	dto, err := h.appService.CancelOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel order: %v", err)
	}

	return h.toProtoResponse(dto), nil
}

func (h *GRPCHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	dto, err := h.appService.GetOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	return h.toProtoResponse(dto), nil
}

func (h *GRPCHandler) toProtoResponse(dto *application.OrderDTO) *pb.OrderResponse {
	return &pb.OrderResponse{
		OrderId:        dto.OrderID,
		UserId:         dto.UserID,
		Symbol:         dto.Symbol,
		Side:           dto.Side,
		OrderType:      dto.OrderType,
		Price:          dto.Price,
		Quantity:       dto.Quantity,
		FilledQuantity: dto.FilledQuantity,
		Status:         dto.Status,
		TimeInForce:    dto.TimeInForce,
		CreatedAt:      dto.CreatedAt,
		UpdatedAt:      dto.UpdatedAt,
	}
}
