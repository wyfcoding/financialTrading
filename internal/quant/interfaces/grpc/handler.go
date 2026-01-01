// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/goapi/quant/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与量化策略和回测相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedQuantServiceServer
	app *application.QuantService // 量化应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的量化应用服务
func NewGRPCHandler(app *application.QuantService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// CreateStrategy 创建策略
// 处理 gRPC CreateStrategy 请求
func (h *GRPCHandler) CreateStrategy(ctx context.Context, req *pb.CreateStrategyRequest) (*pb.CreateStrategyResponse, error) {
	// 调用应用服务创建策略
	id, err := h.app.CreateStrategy(ctx, req.Name, req.Description, req.Script)
	if err != nil {
		slog.Error("Failed to create strategy", "name", req.Name, "error", err)
		return nil, err
	}

	return &pb.CreateStrategyResponse{
		StrategyId: id,
	}, nil
}

// GetStrategy 获取策略
func (h *GRPCHandler) GetStrategy(ctx context.Context, req *pb.GetStrategyRequest) (*pb.GetStrategyResponse, error) {
	strategy, err := h.app.GetStrategy(ctx, req.Id)
	if err != nil {
		slog.Error("Failed to get strategy", "id", req.Id, "error", err)
		return nil, err
	}
	if strategy == nil {
		return &pb.GetStrategyResponse{}, nil
	}

	return &pb.GetStrategyResponse{
		Strategy: toProtoStrategy(strategy),
	}, nil
}

// RunBacktest 运行回测
func (h *GRPCHandler) RunBacktest(ctx context.Context, req *pb.RunBacktestRequest) (*pb.RunBacktestResponse, error) {
	id, err := h.app.RunBacktest(ctx, req.StrategyId, req.Symbol, req.StartTime.AsTime(), req.EndTime.AsTime(), req.InitialCapital)
	if err != nil {
		slog.Error("Failed to run backtest", "strategy_id", req.StrategyId, "symbol", req.Symbol, "error", err)
		return nil, err
	}

	return &pb.RunBacktestResponse{
		BacktestId: id,
	}, nil
}

// GetBacktestResult 获取回测结果
func (h *GRPCHandler) GetBacktestResult(ctx context.Context, req *pb.GetBacktestResultRequest) (*pb.GetBacktestResultResponse, error) {
	result, err := h.app.GetBacktestResult(ctx, req.Id)
	if err != nil {
		slog.Error("Failed to get backtest result", "id", req.Id, "error", err)
		return nil, err
	}
	if result == nil {
		return &pb.GetBacktestResultResponse{}, nil
	}

	return &pb.GetBacktestResultResponse{
		Result: toProtoBacktestResult(result),
	}, nil
}

func toProtoStrategy(s *domain.Strategy) *pb.Strategy {
	return &pb.Strategy{
		Id:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Script:      s.Script,
		Status:      string(s.Status),
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
}

func toProtoBacktestResult(r *domain.BacktestResult) *pb.BacktestResult {
	// 统一处理 ok，虽然这里因为 protobuf 定义为 float 只能接收 float
	// 但避免使用 _ 处理返回值。
	tr, ok1 := r.TotalReturn.Float64()
	md, ok2 := r.MaxDrawdown.Float64()
	sr, ok3 := r.SharpeRatio.Float64()
	if !ok1 || !ok2 || !ok3 {
		slog.Warn("Failed to convert some backtest metrics to float64", "backtest_id", r.ID, "ok1", ok1, "ok2", ok2, "ok3", ok3)
	}

	return &pb.BacktestResult{
		Id:          r.ID,
		StrategyId:  r.StrategyID,
		Symbol:      r.Symbol,
		StartTime:   timestamppb.New(time.UnixMilli(r.StartTime)),
		EndTime:     timestamppb.New(time.UnixMilli(r.EndTime)),
		TotalReturn: tr,
		MaxDrawdown: md,
		SharpeRatio: sr,
		TotalTrades: int32(r.TotalTrades),
		Status:      string(r.Status),
		CreatedAt:   timestamppb.New(r.CreatedAt),
	}
}
