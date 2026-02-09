package interfaces

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/sor/application"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SORHandler SOR gRPC 处理程序
type SORHandler struct {
	pb.UnimplementedSOREngineServiceServer
	appService *application.SORApplicationService
}

func NewSORHandler(appService *application.SORApplicationService) *SORHandler {
	return &SORHandler{
		appService: appService,
	}
}

func (h *SORHandler) CreateSORPlan(ctx context.Context, req *pb.CreateSORPlanRequest) (*pb.CreateSORPlanResponse, error) {
	cmd := application.CreateSORPlanCommand{
		Symbol:   req.Symbol,
		Side:     req.Side,
		Quantity: req.Quantity,
	}

	plan, err := h.appService.CreateSORPlan(ctx, cmd)
	if err != nil {
		return nil, err
	}

	routes := make([]*pb.OrderRoute, 0, len(plan.Routes))
	for _, r := range plan.Routes {
		routes = append(routes, &pb.OrderRoute{
			Exchange: r.Exchange,
			Price:    r.Price,
			Quantity: r.Quantity,
		})
	}

	return &pb.CreateSORPlanResponse{
		Symbol:       plan.Symbol,
		TotalQty:     plan.TotalQty,
		Routes:       routes,
		AveragePrice: plan.AveragePrice,
		GeneratedAt:  timestamppb.New(plan.GeneratedAt),
	}, nil
}

func (h *SORHandler) AggregateDepths(ctx context.Context, req *pb.AggregateDepthsRequest) (*pb.AggregateDepthsResponse, error) {
	depths, err := h.appService.GetDepths(ctx, req.Symbol)
	if err != nil {
		return nil, err
	}

	pbDepths := make([]*pb.MarketDepth, 0, len(depths))
	for _, d := range depths {
		bids := make([]*pb.PriceLevel, 0, len(d.Bids))
		for _, b := range d.Bids {
			bids = append(bids, &pb.PriceLevel{Price: b.Price, Quantity: b.Quantity})
		}
		asks := make([]*pb.PriceLevel, 0, len(d.Asks))
		for _, a := range d.Asks {
			asks = append(asks, &pb.PriceLevel{Price: a.Price, Quantity: a.Quantity})
		}
		pbDepths = append(pbDepths, &pb.MarketDepth{
			Exchange: d.Exchange,
			Symbol:   d.Symbol,
			Bids:     bids,
			Asks:     asks,
		})
	}

	return &pb.AggregateDepthsResponse{
		Depths: pbDepths,
	}, nil
}
