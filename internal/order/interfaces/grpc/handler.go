package grpc

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	pb.UnimplementedOrderServiceServer
	cmd   *application.OrderCommandService
	query *application.OrderQueryService
}

func NewHandler(cmd *application.OrderCommandService, query *application.OrderQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

func (h *Handler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	cmd := application.PlaceOrderCommand{
		UserID:          req.UserId,
		Symbol:          req.Symbol,
		Side:            req.Side.String(),
		Type:            req.Type.String(),
		Price:           req.Price,
		Quantity:        req.Quantity,
		StopPrice:       req.StopPrice,
		TakeProfitPrice: req.TakeProfitPrice,
		ParentOrderID:   req.ParentOrderId,
		OcoOrderID:      req.OcoOrderId,
		IsOCO:           req.IsOco,
	}

	orderID, err := h.cmd.PlaceOrder(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "place order failed: %v", err)
	}

	return &pb.CreateOrderResponse{
		OrderId: orderID,
		Status:  pb.OrderStatus_PENDING,
	}, nil
}

func (h *Handler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	cmd := application.CancelOrderCommand{
		OrderID: req.OrderId,
		UserID:  req.UserId,
		Reason:  "user request",
	}

	if err := h.cmd.CancelOrder(ctx, cmd); err != nil {
		return &pb.CancelOrderResponse{Success: false}, status.Errorf(codes.Internal, "cancel order failed: %v", err)
	}

	return &pb.CancelOrderResponse{Success: true}, nil
}

func (h *Handler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	dto, err := h.query.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get order failed: %v", err)
	}
	if dto == nil {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	return &pb.GetOrderResponse{
		Order: h.toProtoOrder(dto),
	}, nil
}

func (h *Handler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	var statusVal domain.OrderStatus
	if req.Status != "" {
		statusVal = domain.OrderStatus(req.Status)
	}

	dtos, _, err := h.query.ListOrders(ctx, req.UserId, statusVal, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list orders failed: %v", err)
	}

	orders := make([]*pb.Order, 0, len(dtos))
	for _, dto := range dtos {
		orders = append(orders, h.toProtoOrder(dto))
	}

	return &pb.ListOrdersResponse{
		Orders: orders,
	}, nil
}

func (h *Handler) toProtoOrder(d *application.OrderDTO) *pb.Order {
	price, _ := decimal.NewFromString(d.Price)
	qty, _ := decimal.NewFromString(d.Quantity)
	filled, _ := decimal.NewFromString(d.FilledQuantity)

	var side pb.OrderSide
	if d.Side == "buy" {
		side = pb.OrderSide_BUY
	} else {
		side = pb.OrderSide_SELL
	}

	var oType pb.OrderType
	if d.OrderType == "limit" {
		oType = pb.OrderType_LIMIT
	} else {
		oType = pb.OrderType_MARKET
	}

	var status pb.OrderStatus
	switch d.Status {
	case "pending":
		status = pb.OrderStatus_PENDING
	case "validated":
		status = pb.OrderStatus_VALIDATED
	case "filled":
		status = pb.OrderStatus_FILLED
	case "partially_filled":
		status = pb.OrderStatus_PARTIALLY_FILLED
	case "cancelled":
		status = pb.OrderStatus_CANCELLED
	default:
		status = pb.OrderStatus_STATUS_UNSPECIFIED
	}

	stopPrice, _ := decimal.NewFromString(d.StopPrice)
	tpPrice, _ := decimal.NewFromString(d.TakeProfitPrice)

	return &pb.Order{
		Id:              d.OrderID,
		UserId:          d.UserID,
		Symbol:          d.Symbol,
		Side:            side,
		Type:            oType,
		Price:           price.InexactFloat64(),
		Quantity:        qty.InexactFloat64(),
		FilledQuantity:  filled.InexactFloat64(),
		Status:          status,
		StopPrice:       stopPrice.InexactFloat64(),
		TakeProfitPrice: tpPrice.InexactFloat64(),
		ParentOrderId:   d.ParentOrderID,
		OcoOrderId:      d.OcoOrderID,
		IsOco:           d.IsOCO,
		CreatedAt:       timestamppb.New(time.Unix(d.CreatedAt, 0)),
		UpdatedAt:       timestamppb.New(time.Unix(d.UpdatedAt, 0)),
	}
}
