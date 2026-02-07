// Package grpc gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/marketsimulation/v1"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与市场模拟相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedMarketSimulationServiceServer
	cmd   *application.MarketSimulationCommandService
	query *application.MarketSimulationQueryService
}

// NewHandler 创建 gRPC 处理器实例
func NewHandler(cmd *application.MarketSimulationCommandService, query *application.MarketSimulationQueryService) *Handler {
	return &Handler{
		cmd:   cmd,
		query: query,
	}
}

// StartSimulation 启动模拟
func (h *Handler) StartSimulation(ctx context.Context, req *pb.StartSimulationRequest) (*pb.StartSimulationResponse, error) {
	start := time.Now()
	slog.Info("gRPC StartSimulation received", "name", req.Name, "symbol", req.Symbol)

	// Create simulation config first
	cmd := application.CreateSimulationCommand{
		Name:       req.Name,
		Symbol:     req.Symbol,
		Type:       req.Type,
		Parameters: req.Parameters,
	}

	if cmd.Name == "" || cmd.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "name and symbol are required")
	}

	simDTO, err := h.cmd.CreateSimulation(ctx, cmd)
	if err != nil {
		slog.Error("gRPC CreateSimulationConfig failed", "error", err)
		return nil, status.Errorf(codes.Internal, "create simulation failed: %v", err)
	}

	scenarioID := simDTO.ScenarioID

	// Then start it
	err = h.cmd.StartSimulation(ctx, application.StartSimulationCommand{ScenarioID: scenarioID})
	if err != nil {
		slog.Error("gRPC StartSimulation failed", "error", err)
		return nil, status.Errorf(codes.Internal, "start simulation failed: %v", err)
	}

	slog.Info("gRPC StartSimulation successful", "simulation_id", scenarioID, "duration", time.Since(start))
	return &pb.StartSimulationResponse{
		SimulationId: scenarioID,
	}, nil
}

// StopSimulation 停止模拟
func (h *Handler) StopSimulation(ctx context.Context, req *pb.StopSimulationRequest) (*pb.StopSimulationResponse, error) {
	start := time.Now()
	slog.Info("gRPC StopSimulation received", "simulation_id", req.SimulationId)

	err := h.cmd.StopSimulation(ctx, application.StopSimulationCommand{ScenarioID: req.SimulationId})
	success := err == nil

	if err != nil {
		slog.Error("gRPC StopSimulation failed", "simulation_id", req.SimulationId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "stop simulation failed: %v", err)
	}

	slog.Info("gRPC StopSimulation successful", "simulation_id", req.SimulationId, "success", success, "duration", time.Since(start))
	return &pb.StopSimulationResponse{
		Success: success,
	}, nil
}

// GetSimulationStatus 获取模拟状态
func (h *Handler) GetSimulationStatus(ctx context.Context, req *pb.GetSimulationStatusRequest) (*pb.GetSimulationStatusResponse, error) {
	simDTO, err := h.query.GetSimulation(ctx, req.SimulationId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get simulation failed: %v", err)
	}
	if simDTO == nil {
		return &pb.GetSimulationStatusResponse{}, nil
	}

	return &pb.GetSimulationStatusResponse{
		Scenario: toProtoScenario(simDTO),
	}, nil
}

func toProtoScenario(s *application.SimulationDTO) *pb.SimulationScenario {
	return &pb.SimulationScenario{
		Id:          s.ScenarioID,
		Name:        s.Name,
		Description: s.Description,
		Symbol:      s.Symbol,
		Type:        s.Type,
		Parameters:  s.Parameters,
		Status:      s.Status,
		StartTime:   timestamppb.New(s.StartTime),
		EndTime:     timestamppb.New(s.EndTime),
	}
}
