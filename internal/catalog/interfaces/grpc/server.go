package grpc

import (
	"context"
	"fmt"

	v1 "github.com/wyfcoding/financialtrading/go-api/catalog/v1"
	"github.com/wyfcoding/financialtrading/internal/catalog/application"
	"google.golang.org/grpc"
)

type Server struct {
	v1.UnimplementedCatalogServiceServer
	app *application.CatalogApplicationService
}

func NewServer(s *grpc.Server, app *application.CatalogApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterCatalogServiceServer(s, srv)
	return srv
}

func (s *Server) GetProduct(ctx context.Context, req *v1.GetProductRequest) (*v1.GetProductResponse, error) {
	var id uint
	fmt.Sscanf(req.Id, "%d", &id)
	p, err := s.app.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetProductResponse{Product: &v1.Product{Id: fmt.Sprint(p.ID), Name: p.Name, Description: p.Description, Price: p.Price, Stock: int32(p.Stock), Category: p.Category}}, nil
}

func (s *Server) ListProducts(ctx context.Context, req *v1.ListProductsRequest) (*v1.ListProductsResponse, error) {
	products, total, err := s.app.ListProducts(ctx, req.Category, int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}
	var items []*v1.Product
	for _, p := range products {
		items = append(items, &v1.Product{Id: fmt.Sprint(p.ID), Name: p.Name, Description: p.Description, Price: p.Price, Stock: int32(p.Stock), Category: p.Category})
	}
	return &v1.ListProductsResponse{Products: items, Total: int32(total)}, nil
}

func (s *Server) CreateProduct(ctx context.Context, req *v1.CreateProductRequest) (*v1.CreateProductResponse, error) {
	id, err := s.app.CreateProduct(ctx, req.Name, req.Description, req.Price, int(req.Stock), req.Category)
	if err != nil {
		return nil, err
	}
	return &v1.CreateProductResponse{Id: fmt.Sprint(id)}, nil
}
