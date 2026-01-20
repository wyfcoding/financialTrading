package grpc

import (
	"context"
	"time"

	v1 "github.com/wyfcoding/financialtrading/go-api/pricing/v1"
	"github.com/wyfcoding/financialtrading/internal/pricing/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedPricingServiceServer
	app *application.PricingApplicationService
}

func NewServer(s *grpc.Server, app *application.PricingApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterPricingServiceServer(s, srv)
	return srv
}

func (s *Server) GetPrice(ctx context.Context, req *v1.GetPriceRequest) (*v1.GetPriceResponse, error) {
	dto, err := s.app.GetPrice(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetPriceResponse{Price: s.toProto(dto)}, nil
}

func (s *Server) ListPrices(ctx context.Context, req *v1.ListPricesRequest) (*v1.ListPricesResponse, error) {
	dtos, err := s.app.ListPrices(ctx, req.Symbols)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbPrices []*v1.Price
	for _, d := range dtos {
		pbPrices = append(pbPrices, s.toProto(d))
	}
	return &v1.ListPricesResponse{Prices: pbPrices}, nil
}

// SubscribePrices implements a simple polling stream for demonstration.
// In production, this would hook into an event bus or channel.
func (s *Server) SubscribePrices(req *v1.SubscribePricesRequest, stream v1.PricingService_SubscribePricesServer) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			dtos, err := s.app.ListPrices(context.Background(), req.Symbols)
			if err != nil {
				continue
			}
			for _, d := range dtos {
				if err := stream.Send(&v1.PriceUpdate{Price: s.toProto(d)}); err != nil {
					return err
				}
			}
		}
	}
}

func (s *Server) toProto(d *application.PriceDTO) *v1.Price {
	return &v1.Price{
		Symbol:    d.Symbol,
		Bid:       d.Bid,
		Ask:       d.Ask,
		Mid:       d.Mid,
		Source:    d.Source,
		Timestamp: timestamppb.New(d.Timestamp),
	}
}
