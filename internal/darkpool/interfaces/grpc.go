package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/darkpool/v1"
	"github.com/wyfcoding/financialtrading/internal/darkpool/application"
)

type DarkpoolHandler struct {
	pb.UnimplementedDarkpoolServiceServer
	app *application.DarkpoolService
}

func NewDarkpoolHandler(app *application.DarkpoolService) *DarkpoolHandler {
	return &DarkpoolHandler{app: app}
}

func (h *DarkpoolHandler) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.PlaceOrderResponse, error) {
	return h.app.PlaceOrder(ctx, req)
}

func (h *DarkpoolHandler) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	return h.app.CancelOrder(ctx, req.OrderId, req.UserId)
}

func (h *DarkpoolHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	return h.app.GetOrder(ctx, req.OrderId)
}

func (h *DarkpoolHandler) ListUserOrders(ctx context.Context, req *pb.ListUserOrdersRequest) (*pb.ListUserOrdersResponse, error) {
	return h.app.ListUserOrders(ctx, req.UserId, req.Status)
}
