package grpc

import (
	"context"
	"math"
	"strings"

	pb "github.com/wyfcoding/financialtrading/go-api/sor/v1"
	"github.com/wyfcoding/financialtrading/internal/sor/application"
	"github.com/wyfcoding/financialtrading/internal/sor/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedSOREngineServiceServer
	app *application.SORApplicationService
}

func NewServer(app *application.SORApplicationService) *Server {
	return &Server{
		app: app,
	}
}

func (s *Server) CreateSORPlan(ctx context.Context, req *pb.CreateSORPlanRequest) (*pb.CreateSORPlanResponse, error) {
	side := domain.SideBuy
	if strings.EqualFold(req.Side, "SELL") {
		side = domain.SideSell
	}
	cmd := application.CreateSORPlanCommand{
		Symbol:   req.Symbol,
		Side:     side,
		Quantity: req.Quantity,
	}

	plan, err := s.app.CreateSORPlan(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create SOR plan: %v", err)
	}

	routes := make([]*pb.OrderRoute, 0, len(plan.Routes))
	for _, r := range plan.Routes {
		routes = append(routes, &pb.OrderRoute{
			Exchange: r.VenueID,
			Price:    int64(math.Round(r.Price)),
			Quantity: r.Quantity,
		})
	}

	return &pb.CreateSORPlanResponse{
		Symbol:       plan.Symbol,
		TotalQty:     plan.TotalQuantity,
		Routes:       routes,
		AveragePrice: int64(math.Round(plan.AveragePrice)),
		GeneratedAt:  timestamppb.New(plan.GeneratedAt),
	}, nil
}

func (s *Server) AggregateDepths(ctx context.Context, req *pb.AggregateDepthsRequest) (*pb.AggregateDepthsResponse, error) {
	depths, err := s.app.GetDepths(ctx, application.GetDepthsQuery{Symbol: req.Symbol})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to aggregate depths: %v", err)
	}

	pbDepths := make([]*pb.MarketDepth, 0, len(depths))
	for _, d := range depths {
		bids := make([]*pb.PriceLevel, 0, len(d.Bids))
		for _, b := range d.Bids {
			bids = append(bids, &pb.PriceLevel{Price: int64(math.Round(b.Price)), Quantity: b.Quantity})
		}

		asks := make([]*pb.PriceLevel, 0, len(d.Asks))
		for _, a := range d.Asks {
			asks = append(asks, &pb.PriceLevel{Price: int64(math.Round(a.Price)), Quantity: a.Quantity})
		}

		pbDepths = append(pbDepths, &pb.MarketDepth{
			Exchange: d.VenueID,
			Symbol:   d.Symbol,
			Bids:     bids,
			Asks:     asks,
		})
	}

	return &pb.AggregateDepthsResponse{
		Depths: pbDepths,
	}, nil
}
