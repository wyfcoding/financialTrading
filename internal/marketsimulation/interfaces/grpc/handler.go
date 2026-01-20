// Package grpc gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/marketsimulation/v1"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与市场模拟相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedMarketSimulationServiceServer
	app *application.MarketSimulationApplicationService
}

// NewHandler 创建 gRPC 处理器实例
func NewHandler(app *application.MarketSimulationApplicationService) *Handler {
	return &Handler{
		app: app,
	}
}

// StartSimulation 启动模拟
func (h *Handler) StartSimulation(ctx context.Context, req *pb.StartSimulationRequest) (*pb.StartSimulationResponse, error) {
	start := time.Now()
	slog.Info("gRPC StartSimulation received", "name", req.Name, "symbol", req.Symbol)

	// Create simulation config first
	// Note: proto lacks InitialPrice etc, using defaults or parsing from Parameters
	cmd := application.CreateSimulationCommand{
		Name:         req.Name,
		Symbol:       req.Symbol,
		InitialPrice: 100.0, // Default
		Volatility:   0.2,   // Default
		Drift:        0.05,  // Default
		IntervalMs:   1000,  // Default
	}

	simDTO, err := h.app.CreateSimulationConfig(ctx, cmd)
	if err != nil {
		slog.Error("gRPC CreateSimulationConfig failed", "error", err)
		return nil, err
	}

	// Use ScenarioID from domain (via DTO)
	// DTO.ID is uint from gorm.Model, DTO needs ScenarioID string? Let's work with what we have.
	// Actually we need to get the string ScenarioID from the created simulation.
	// But our DTO has uint ID. We need to either:
	// 1. Add ScenarioID to DTO
	// 2. Use strconv
	// For now, start using the ID to get, which internally uses ScenarioID.
	// Since SDK returns DTO with uint ID, we likely need to store ScenarioID in DTO.
	// Will update DTO. But for now, let's just use the ID converted to string.

	scenarioID := simDTO.ScenarioID

	// Then start it
	err = h.app.StartSimulation(ctx, scenarioID)
	if err != nil {
		slog.Error("gRPC StartSimulation failed", "error", err)
		return nil, err
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

	err := h.app.StopSimulation(ctx, req.SimulationId)
	success := err == nil

	if err != nil {
		slog.Error("gRPC StopSimulation failed", "simulation_id", req.SimulationId, "error", err, "duration", time.Since(start))
		return nil, err
	}

	slog.Info("gRPC StopSimulation successful", "simulation_id", req.SimulationId, "success", success, "duration", time.Since(start))
	return &pb.StopSimulationResponse{
		Success: success,
	}, nil
}

// GetSimulationStatus 获取模拟状态
func (h *Handler) GetSimulationStatus(ctx context.Context, req *pb.GetSimulationStatusRequest) (*pb.GetSimulationStatusResponse, error) {
	simDTO, err := h.app.GetSimulation(ctx, req.SimulationId)
	if err != nil {
		return nil, err
	}
	if simDTO == nil {
		return &pb.GetSimulationStatusResponse{}, nil
	}

	return &pb.GetSimulationStatusResponse{
		Scenario: &pb.SimulationScenario{
			Id:     simDTO.ScenarioID,
			Name:   simDTO.Name,
			Symbol: simDTO.Symbol,
			Status: simDTO.Status,
		},
	}, nil
}

func toProtoScenario(s *domain.Simulation) *pb.SimulationScenario {
	return &pb.SimulationScenario{
		Id:          s.ScenarioID,
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
