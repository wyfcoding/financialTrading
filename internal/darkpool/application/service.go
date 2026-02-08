package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/darkpool/v1"
	"github.com/wyfcoding/financialtrading/internal/darkpool/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DarkpoolService struct {
	repo domain.DarkpoolRepository
}

func NewDarkpoolService(repo domain.DarkpoolRepository) *DarkpoolService {
	return &DarkpoolService{repo: repo}
}

func (s *DarkpoolService) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.PlaceOrderResponse, error) {
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return nil, err
	}
	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, err
	}

	orderID := fmt.Sprintf("dark_%d", time.Now().UnixNano())
	order := &domain.DarkOrder{
		OrderID:        orderID,
		UserID:         req.UserId,
		Symbol:         req.Symbol,
		Side:           domain.OrderSide(req.Side),
		Price:          price,
		Quantity:       quantity,
		FilledQuantity: decimal.Zero,
		Status:         domain.StatusNew,
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}

	return &pb.PlaceOrderResponse{
		OrderId: orderID,
		Status:  string(domain.StatusNew),
	}, nil
}

func (s *DarkpoolService) CancelOrder(ctx context.Context, orderID, userID string) (*pb.CancelOrderResponse, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("order %s not found", orderID)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	order.Status = domain.StatusCancelled
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}

	return &pb.CancelOrderResponse{Success: true}, nil
}

func (s *DarkpoolService) GetOrder(ctx context.Context, orderID string) (*pb.GetOrderResponse, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	return &pb.GetOrderResponse{
		Order: mapToPb(order),
	}, nil
}

func (s *DarkpoolService) ListUserOrders(ctx context.Context, userID, status string) (*pb.ListUserOrdersResponse, error) {
	orders, err := s.repo.ListOrders(ctx, userID, status)
	if err != nil {
		return nil, err
	}

	var pbOrders []*pb.DarkOrder
	for _, o := range orders {
		pbOrders = append(pbOrders, mapToPb(o))
	}

	return &pb.ListUserOrdersResponse{Orders: pbOrders}, nil
}

func mapToPb(o *domain.DarkOrder) *pb.DarkOrder {
	return &pb.DarkOrder{
		OrderId:        o.OrderID,
		UserId:         o.UserID,
		Symbol:         o.Symbol,
		Side:           string(o.Side),
		Price:          o.Price.String(),
		Quantity:       o.Quantity.String(),
		FilledQuantity: o.FilledQuantity.String(),
		Status:         string(o.Status),
		CreatedAt:      timestamppb.New(o.CreatedAt),
	}
}
