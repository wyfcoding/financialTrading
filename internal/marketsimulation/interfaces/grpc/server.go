package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/marketsimulation/v1"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedMarketSimulationServer
	app *application.MarketSimulationApplicationService
}

func NewServer(s *grpc.Server, app *application.MarketSimulationApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterMarketSimulationServer(s, srv)
	return srv
}

func (s *Server) CreateSimulation(ctx context.Context, req *v1.CreateSimulationRequest) (*v1.CreateSimulationResponse, error) {
	cmd := application.CreateSimulationCommand{
		Name:         req.Name,
		Symbol:       req.Symbol,
		InitialPrice: req.InitialPrice,
		Volatility:   req.Volatility,
		Drift:        req.Drift,
		IntervalMs:   int(req.IntervalMs),
	}

	dto, err := s.app.CreateSimulationConfig(ctx, cmd)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.CreateSimulationResponse{
		SimulationId: uint32(dto.ID),
	}, nil
}

func (s *Server) StartSimulation(ctx context.Context, req *v1.StartSimulationRequest) (*v1.StartSimulationResponse, error) {
	err := s.app.StartSimulation(ctx, uint(req.SimulationId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.StartSimulationResponse{Success: true}, nil
}

func (s *Server) StopSimulation(ctx context.Context, req *v1.StopSimulationRequest) (*v1.StopSimulationResponse, error) {
	err := s.app.StopSimulation(ctx, uint(req.SimulationId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.StopSimulationResponse{Success: true}, nil
}

func (s *Server) ListSimulations(ctx context.Context, req *v1.ListSimulationsRequest) (*v1.ListSimulationsResponse, error) {
	dtos, err := s.app.ListSimulations(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var sims []*v1.Simulation
	for _, d := range dtos {
		sims = append(sims, &v1.Simulation{
			Id:           uint32(d.ID),
			Name:         d.Name,
			Symbol:       d.Symbol,
			InitialPrice: d.InitialPrice,
			Volatility:   d.Volatility,
			Drift:        d.Drift,
			IntervalMs:   int32(d.IntervalMs),
			Status:       d.Status,
		})
	}

	return &v1.ListSimulationsResponse{Simulations: sims}, nil
}
