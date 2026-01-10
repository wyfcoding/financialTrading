// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/goapi/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler gRPC 处理器
// 负责处理与订单相关的 gRPC 请求
type Handler struct {
	// 嵌入 UnimplementedOrderServiceServer 以实现向前兼容
	pb.UnimplementedOrderServiceServer
	// 订单应用服务，处理业务逻辑
	service *application.OrderService
}

// NewHandler 创建 gRPC 处理器实例
// service: 注入的订单应用服务
func NewHandler(service *application.OrderService) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateOrder 创建订单
// 处理 gRPC CreateOrder 请求
func (h *Handler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	start := time.Now()
	slog.Info("gRPC CreateOrder received", "user_id", req.UserId, "symbol", req.Symbol, "side", req.Side, "price", req.Price, "quantity", req.Quantity)

	// 调用应用服务创建订单
	dto, err := h.service.CreateOrder(ctx, &application.CreateOrderRequest{
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
		slog.Error("gRPC CreateOrder failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	slog.Info("gRPC CreateOrder successful", "order_id", dto.OrderID, "user_id", req.UserId, "duration", time.Since(start))
	// 返回创建成功的订单
	return &pb.CreateOrderResponse{
		Order: h.toProtoOrder(dto),
	}, nil
}

// CancelOrder 取消订单
// 处理 gRPC CancelOrder 请求
func (h *Handler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	start := time.Now()
	slog.Info("gRPC CancelOrder received", "order_id", req.OrderId, "user_id", req.UserId)

	// 调用应用服务取消订单
	dto, err := h.service.CancelOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		slog.Error("gRPC CancelOrder failed", "order_id", req.OrderId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to cancel order: %v", err)
	}

	slog.Info("gRPC CancelOrder successful", "order_id", req.OrderId, "duration", time.Since(start))
	return &pb.CancelOrderResponse{
		Order: h.toProtoOrder(dto),
	}, nil
}

// GetOrder 获取订单详情
// 处理 gRPC GetOrder 请求
func (h *Handler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	dto, err := h.service.GetOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	return &pb.GetOrderResponse{
		Order: h.toProtoOrder(dto),
	}, nil
}

// ListOrders 分页检索订单列表
func (h *Handler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 100 // 恢复模式下可能需要更大量拉取
	}
	offset := max(int((req.Page-1)*req.PageSize), 0)

	dtos, total, err := h.service.ListOrders(ctx, req.UserId, req.Symbol, domain.OrderStatus(req.Status), limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}

	orders := make([]*pb.Order, 0, len(dtos))
	for _, dto := range dtos {
		orders = append(orders, h.toProtoOrder(dto))
	}

	return &pb.ListOrdersResponse{
		Orders:   orders,
		Total:    total,
		Page:     req.Page,
		PageSize: int32(limit),
	}, nil
}

func (h *Handler) toProtoOrder(dto *application.OrderDTO) *pb.Order {
	return &pb.Order{
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
