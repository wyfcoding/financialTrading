package grpc

import (
	"context"
	"math"
	"strconv"
	"strings"
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
		UserID:        req.UserId,
		Symbol:        req.Symbol,
		Side:          orderSideToDomain(req.Side),
		Type:          orderTypeToDomain(req.Type),
		Price:         req.Price,
		Quantity:      req.Quantity,
		TimeInForce:   timeInForceToDomain(req.TimeInForce),
		ParentOrderID: req.ParentOrderId,
	}
	applyCompatMetadata(&cmd, req.Metadata)

	orderID, err := h.cmd.PlaceOrder(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "place order failed: %v", err)
	}

	return &pb.CreateOrderResponse{
		OrderId:       orderID,
		ClientOrderId: req.ClientOrderId,
		Status:        pb.OrderStatus_ORDER_STATUS_PENDING_NEW,
		CreatedAt:     timestamppb.Now(),
	}, nil
}

func (h *Handler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	cmd := application.CancelOrderCommand{
		OrderID: req.OrderId,
		UserID:  req.UserId,
		Reason:  req.Reason,
	}
	if cmd.Reason == "" {
		cmd.Reason = "user request"
	}

	if err := h.cmd.CancelOrder(ctx, cmd); err != nil {
		return &pb.CancelOrderResponse{Success: false}, status.Errorf(codes.Internal, "cancel order failed: %v", err)
	}

	return &pb.CancelOrderResponse{
		Success:     true,
		FinalStatus: pb.OrderStatus_ORDER_STATUS_CANCELLED,
		CancelledAt: timestamppb.Now(),
	}, nil
}

func (h *Handler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	orderID := req.OrderId
	if orderID == "" {
		orderID = req.ClientOrderId
	}
	dto, err := h.query.GetOrder(ctx, orderID)
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
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := int((page - 1) * pageSize)

	statusVal := orderStatusToDomain(req.Status)
	dtos, total, err := h.query.ListOrders(ctx, req.UserId, statusVal, int(pageSize), offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list orders failed: %v", err)
	}

	orders := make([]*pb.Order, 0, len(dtos))
	for _, dto := range dtos {
		orders = append(orders, h.toProtoOrder(dto))
	}

	return &pb.ListOrdersResponse{
		Orders:     orders,
		TotalCount: int64ToInt32(total),
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func (h *Handler) toProtoOrder(d *application.OrderDTO) *pb.Order {
	price := parseDecimalString(d.Price)
	qty := parseDecimalString(d.Quantity)
	filled := parseDecimalString(d.FilledQuantity)
	avg := parseDecimalString(d.AveragePrice)
	remaining := qty - filled
	if remaining < 0 {
		remaining = 0
	}

	metadata := make(map[string]string)
	if d.StopPrice != "" && d.StopPrice != "0" {
		metadata["stop_price"] = d.StopPrice
	}
	if d.TakeProfitPrice != "" && d.TakeProfitPrice != "0" {
		metadata["take_profit_price"] = d.TakeProfitPrice
	}
	if d.IsOCO {
		metadata["is_oco"] = "true"
	}
	if len(metadata) == 0 {
		metadata = nil
	}

	linkedType := pb.LinkedOrderType_LINKED_UNSPECIFIED
	if d.IsOCO || d.OcoOrderID != "" {
		linkedType = pb.LinkedOrderType_OCO
	}

	return &pb.Order{
		Id:                d.OrderID,
		UserId:            d.UserID,
		Symbol:            d.Symbol,
		Side:              domainSideToOrderSide(d.Side),
		Type:              domainTypeToOrderType(d.OrderType),
		Price:             price,
		Quantity:          qty,
		FilledQuantity:    filled,
		RemainingQuantity: remaining,
		AveragePrice:      avg,
		TotalValue:        avg * filled,
		Status:            domainStatusToOrderStatus(d.Status),
		TimeInForce:       domainTimeInForceToPB(d.TimeInForce),
		ParentOrderId:     d.ParentOrderID,
		LinkedOrderId:     d.OcoOrderID,
		LinkedType:        linkedType,
		CreatedAt:         unixToTimestamp(d.CreatedAt),
		UpdatedAt:         unixToTimestamp(d.UpdatedAt),
		Metadata:          metadata,
	}
}

func applyCompatMetadata(cmd *application.PlaceOrderCommand, md map[string]string) {
	if len(md) == 0 {
		return
	}
	if v := md["oco_order_id"]; v != "" {
		cmd.OcoOrderID = v
	}
	if v := md["is_oco"]; v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cmd.IsOCO = b
		}
	}
	if v := md["stop_price"]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cmd.StopPrice = f
		}
	}
	if v := md["take_profit_price"]; v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cmd.TakeProfitPrice = f
		}
	}
}

func orderSideToDomain(side pb.OrderSide) string {
	switch side {
	case pb.OrderSide_ORDER_SIDE_BUY:
		return string(domain.SideBuy)
	case pb.OrderSide_ORDER_SIDE_SELL:
		return string(domain.SideSell)
	default:
		return ""
	}
}

func orderTypeToDomain(typ pb.OrderType) string {
	switch typ {
	case pb.OrderType_ORDER_TYPE_LIMIT:
		return string(domain.TypeLimit)
	case pb.OrderType_ORDER_TYPE_MARKET:
		return string(domain.TypeMarket)
	case pb.OrderType_ORDER_TYPE_STOP_LIMIT:
		return string(domain.TypeStopLimit)
	case pb.OrderType_ORDER_TYPE_STOP_MARKET:
		return string(domain.TypeStopMarket)
	case pb.OrderType_ORDER_TYPE_TRAILING_STOP:
		return string(domain.TypeTrailing)
	default:
		name := strings.TrimPrefix(typ.String(), "ORDER_TYPE_")
		return strings.ToLower(name)
	}
}

func timeInForceToDomain(tif pb.TimeInForce) string {
	switch tif {
	case pb.TimeInForce_TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		return string(domain.IOC)
	case pb.TimeInForce_TIME_IN_FORCE_FILL_OR_KILL:
		return string(domain.FOK)
	case pb.TimeInForce_TIME_IN_FORCE_GOOD_TILL_CANCEL, pb.TimeInForce_TIME_IN_FORCE_DAY:
		return string(domain.GTC)
	default:
		return ""
	}
}

func orderStatusToDomain(status pb.OrderStatus) domain.OrderStatus {
	switch status {
	case pb.OrderStatus_ORDER_STATUS_PENDING_NEW:
		return domain.StatusPending
	case pb.OrderStatus_ORDER_STATUS_NEW:
		return domain.StatusValidated
	case pb.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED:
		return domain.StatusPartiallyFilled
	case pb.OrderStatus_ORDER_STATUS_FILLED:
		return domain.StatusFilled
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.StatusCancelled
	case pb.OrderStatus_ORDER_STATUS_REJECTED:
		return domain.StatusRejected
	case pb.OrderStatus_ORDER_STATUS_EXPIRED:
		return domain.StatusExpired
	default:
		return ""
	}
}

func domainSideToOrderSide(side string) pb.OrderSide {
	switch strings.ToLower(side) {
	case "buy":
		return pb.OrderSide_ORDER_SIDE_BUY
	case "sell":
		return pb.OrderSide_ORDER_SIDE_SELL
	default:
		return pb.OrderSide_ORDER_SIDE_UNSPECIFIED
	}
}

func domainTypeToOrderType(orderType string) pb.OrderType {
	switch strings.ToLower(orderType) {
	case "limit":
		return pb.OrderType_ORDER_TYPE_LIMIT
	case "market":
		return pb.OrderType_ORDER_TYPE_MARKET
	case "stop_limit":
		return pb.OrderType_ORDER_TYPE_STOP_LIMIT
	case "stop_market":
		return pb.OrderType_ORDER_TYPE_STOP_MARKET
	case "trailing_stop":
		return pb.OrderType_ORDER_TYPE_TRAILING_STOP
	default:
		return pb.OrderType_ORDER_TYPE_UNSPECIFIED
	}
}

func domainStatusToOrderStatus(status string) pb.OrderStatus {
	switch strings.ToLower(status) {
	case "pending":
		return pb.OrderStatus_ORDER_STATUS_PENDING_NEW
	case "validated":
		return pb.OrderStatus_ORDER_STATUS_NEW
	case "partially_filled":
		return pb.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED
	case "filled":
		return pb.OrderStatus_ORDER_STATUS_FILLED
	case "cancelled":
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	case "rejected":
		return pb.OrderStatus_ORDER_STATUS_REJECTED
	case "expired":
		return pb.OrderStatus_ORDER_STATUS_EXPIRED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func domainTimeInForceToPB(tif string) pb.TimeInForce {
	switch strings.ToUpper(tif) {
	case "IOC":
		return pb.TimeInForce_TIME_IN_FORCE_IMMEDIATE_OR_CANCEL
	case "FOK":
		return pb.TimeInForce_TIME_IN_FORCE_FILL_OR_KILL
	case "GTC":
		return pb.TimeInForce_TIME_IN_FORCE_GOOD_TILL_CANCEL
	default:
		return pb.TimeInForce_TIME_IN_FORCE_UNSPECIFIED
	}
}

func parseDecimalString(v string) float64 {
	d, err := decimal.NewFromString(v)
	if err != nil {
		return 0
	}
	return d.InexactFloat64()
}

func int64ToInt32(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

func unixToTimestamp(sec int64) *timestamppb.Timestamp {
	if sec <= 0 {
		return nil
	}
	return timestamppb.New(time.Unix(sec, 0))
}
