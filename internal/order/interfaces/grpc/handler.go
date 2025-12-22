// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/order"
	"github.com/wyfcoding/financialTrading/internal/order/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与订单相关的 gRPC 请求
type GRPCHandler struct {
	// 嵌入 UnimplementedOrderServiceServer 以实现向前兼容
	pb.UnimplementedOrderServiceServer
	// 订单应用服务，处理业务逻辑
	appService *application.OrderApplicationService
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的订单应用服务
func NewGRPCHandler(appService *application.OrderApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// CreateOrder 创建订单
// 处理 gRPC CreateOrder 请求
func (h *GRPCHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	// 调用应用服务创建订单
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

	// 返回创建成功的订单
	return h.toProtoResponse(dto), nil
}

// CancelOrder 取消订单
// 处理 gRPC CancelOrder 请求
func (h *GRPCHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.OrderResponse, error) {
	// 调用应用服务取消订单
	dto, err := h.appService.CancelOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel order: %v", err)
	}

	return h.toProtoResponse(dto), nil
}

// GetOrder 获取订单详情
// 处理 gRPC GetOrder 请求
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
