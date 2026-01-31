package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/cart/v1"
	"github.com/wyfcoding/financialtrading/internal/cart/application"
	"google.golang.org/grpc"
)

type Server struct {
	v1.UnimplementedCartServiceServer
	app *application.CartApplicationService
}

func NewServer(s *grpc.Server, app *application.CartApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterCartServiceServer(s, srv)
	return srv
}

func (s *Server) GetCart(ctx context.Context, req *v1.GetCartRequest) (*v1.GetCartResponse, error) {
	cart, err := s.app.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	var items []*v1.CartItem
	for _, i := range cart.Items {
		items = append(items, &v1.CartItem{ProductId: i.ProductID, Quantity: int32(i.Quantity), Price: i.Price})
	}
	return &v1.GetCartResponse{UserId: cart.UserID, Items: items, Total: cart.Total()}, nil
}

func (s *Server) AddItem(ctx context.Context, req *v1.AddItemRequest) (*v1.AddItemResponse, error) {
	if err := s.app.AddItem(ctx, req.UserId, req.ProductId, int(req.Quantity), req.Price); err != nil {
		return nil, err
	}
	return &v1.AddItemResponse{Success: true}, nil
}

func (s *Server) RemoveItem(ctx context.Context, req *v1.RemoveItemRequest) (*v1.RemoveItemResponse, error) {
	if err := s.app.RemoveItem(ctx, req.UserId, req.ProductId); err != nil {
		return nil, err
	}
	return &v1.RemoveItemResponse{Success: true}, nil
}

func (s *Server) ClearCart(ctx context.Context, req *v1.ClearCartRequest) (*v1.ClearCartResponse, error) {
	if err := s.app.ClearCart(ctx, req.UserId); err != nil {
		return nil, err
	}
	return &v1.ClearCartResponse{Success: true}, nil
}
