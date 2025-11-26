package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/market-simulation"
	"github.com/wyfcoding/financialTrading/internal/market-simulation/application"
	"github.com/wyfcoding/financialTrading/internal/market-simulation/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
type GRPCHandler struct {
	pb.UnimplementedMarketSimulationServiceServer
	app *application.MarketSimulationService
}

// NewGRPCHandler 创建 gRPC 处理器实例
func NewGRPCHandler(app *application.MarketSimulationService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// StartSimulation 启动模拟
func (h *GRPCHandler) StartSimulation(ctx context.Context, req *pb.StartSimulationRequest) (*pb.StartSimulationResponse, error) {
	id, err := h.app.StartSimulation(ctx, req.Name, req.Symbol, req.Type, req.Parameters)
	if err != nil {
		return nil, err
	}

	return &pb.StartSimulationResponse{
		SimulationId: id,
	}, nil
}

// StopSimulation 停止模拟
func (h *GRPCHandler) StopSimulation(ctx context.Context, req *pb.StopSimulationRequest) (*pb.StopSimulationResponse, error) {
	success, err := h.app.StopSimulation(ctx, req.SimulationId)
	if err != nil {
		return nil, err
	}

	return &pb.StopSimulationResponse{
		Success: success,
	}, nil
}

// GetSimulationStatus 获取模拟状态
func (h *GRPCHandler) GetSimulationStatus(ctx context.Context, req *pb.GetSimulationStatusRequest) (*pb.GetSimulationStatusResponse, error) {
	scenario, err := h.app.GetSimulationStatus(ctx, req.SimulationId)
	if err != nil {
		return nil, err
	}
	if scenario == nil {
		return &pb.GetSimulationStatusResponse{}, nil
	}

	return &pb.GetSimulationStatusResponse{
		Scenario: toProtoScenario(scenario),
	}, nil
}

func toProtoScenario(s *domain.SimulationScenario) *pb.SimulationScenario {
	return &pb.SimulationScenario{
		Id:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Symbol:      s.Symbol,
		Type:        string(s.Type),
		Parameters:  s.Parameters,
		Status:      string(s.Status),
		StartTime:   timestamppb.New(s.StartTime),
		EndTime:     timestamppb.New(s.EndTime),
	}
}
