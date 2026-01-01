// Package grpc  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/goapi/marketsimulation/v1"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/application"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与市场模拟相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedMarketSimulationServiceServer
	app *application.MarketSimulationService // 市场模拟应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的市场模拟应用服务
func NewGRPCHandler(app *application.MarketSimulationService) *GRPCHandler {
	return &GRPCHandler{
		app: app,
	}
}

// StartSimulation 启动模拟
// 处理 gRPC StartSimulation 请求
func (h *GRPCHandler) StartSimulation(ctx context.Context, req *pb.StartSimulationRequest) (*pb.StartSimulationResponse, error) {
	start := time.Now()
	slog.Info("gRPC StartSimulation received", "name", req.Name, "symbol", req.Symbol, "type", req.Type)

	// 调用应用服务启动模拟
	scenarioID, err := h.app.StartSimulation(ctx, req.Name, req.Symbol, req.Type, req.Parameters)
	if err != nil {
		slog.Error("gRPC StartSimulation failed", "name", req.Name, "error", err, "duration", time.Since(start))
		return nil, err
	}

	slog.Info("gRPC StartSimulation successful", "simulation_id", scenarioID, "duration", time.Since(start))
	return &pb.StartSimulationResponse{
		SimulationId: scenarioID,
	}, nil
}

// StopSimulation 停止模拟
func (h *GRPCHandler) StopSimulation(ctx context.Context, req *pb.StopSimulationRequest) (*pb.StopSimulationResponse, error) {
	start := time.Now()
	slog.Info("gRPC StopSimulation received", "simulation_id", req.SimulationId)

	success, err := h.app.StopSimulation(ctx, req.SimulationId)
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
func (h *GRPCHandler) GetSimulationStatus(ctx context.Context, req *pb.GetSimulationStatusRequest) (*pb.GetSimulationStatusResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetSimulationStatus received", "simulation_id", req.SimulationId)

	scenario, err := h.app.GetSimulationStatus(ctx, req.SimulationId)
	if err != nil {
		slog.Error("gRPC GetSimulationStatus failed", "simulation_id", req.SimulationId, "error", err, "duration", time.Since(start))
		return nil, err
	}
	if scenario == nil {
		slog.Debug("gRPC GetSimulationStatus successful (not found)", "simulation_id", req.SimulationId, "duration", time.Since(start))
		return &pb.GetSimulationStatusResponse{}, nil
	}

	slog.Debug("gRPC GetSimulationStatus successful", "simulation_id", req.SimulationId, "status", scenario.Status, "duration", time.Since(start))
	return &pb.GetSimulationStatusResponse{
		Scenario: toProtoScenario(scenario),
	}, nil
}

func toProtoScenario(s *domain.SimulationScenario) *pb.SimulationScenario {
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