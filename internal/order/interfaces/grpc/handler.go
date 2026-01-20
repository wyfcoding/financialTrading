package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	pb.UnimplementedOrderServiceServer
	app   *application.OrderManager
	query *application.OrderQuery
}

func NewHandler(app *application.OrderManager, query *application.OrderQuery) *Handler {
	return &Handler{
		app:   app,
		query: query,
	}
}

func (h *Handler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	createReq := &application.CreateOrderRequest{
		UserID:    req.UserId,
		Symbol:    req.Symbol,
		Side:      "buy",   // default
		OrderType: "limit", // default
		Price:     fmt.Sprintf("%f", req.Price),
		Quantity:  fmt.Sprintf("%f", req.Quantity),
	}

	if req.Side == pb.OrderSide_SELL {
		createReq.Side = "sell"
	}
	if req.Type == pb.OrderType_MARKET {
		createReq.OrderType = "market"
	}

	dto, err := h.app.CreateOrder(ctx, createReq)
	if err != nil {
		return nil, err
	}

	var status pb.OrderStatus
	switch dto.Status {
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

	return &pb.CreateOrderResponse{
		OrderId: dto.OrderID,
		Status:  status,
	}, nil
}

func (h *Handler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	_, err := h.app.CancelOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		return &pb.CancelOrderResponse{Success: false}, err
	}
	return &pb.CancelOrderResponse{Success: true}, nil
}

func (h *Handler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	dto, err := h.query.GetOrder(ctx, req.OrderId)
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
	return &pb.ListOrdersResponse{}, nil
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
