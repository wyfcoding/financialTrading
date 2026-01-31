package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	pb.UnimplementedOrderServiceServer
	app *application.OrderService
}

func NewHandler(app *application.OrderService) *Handler {
	return &Handler{
		app: app,
	}
}

func (h *Handler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	cmd := application.PlaceOrderCommand{
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side.String(),
		Type:     req.Type.String(),
		Price:    req.Price,
		Quantity: req.Quantity,
	}

	orderID, err := h.app.PlaceOrder(ctx, cmd)
	if err != nil {
		return nil, err
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

	err := h.app.CancelOrder(ctx, cmd)
	if err != nil {
		return &pb.CancelOrderResponse{Success: false}, err
	}

	return &pb.CancelOrderResponse{Success: true}, nil
}

func (h *Handler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	dto, err := h.app.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}
	if dto == nil {
		return nil, fmt.Errorf("order not found")
	}
	return &pb.GetOrderResponse{
		Order: h.toProtoOrder(dto),
	}, nil
}

func (h *Handler) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	var status domain.OrderStatus
	if req.Status != "" {
		status = domain.OrderStatus(req.Status)
	}

	dtos, _, err := h.app.ListOrders(ctx, req.UserId, status, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
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

	return &pb.Order{
		Id:             d.OrderID,
		UserId:         d.UserID,
		Symbol:         d.Symbol,
		Side:           side,
		Type:           oType,
		Price:          price.InexactFloat64(),
		Quantity:       qty.InexactFloat64(),
		FilledQuantity: filled.InexactFloat64(),
		Status:         status,
		CreatedAt:      timestamppb.New(time.Unix(d.CreatedAt, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(d.UpdatedAt, 0)),
	}
}
