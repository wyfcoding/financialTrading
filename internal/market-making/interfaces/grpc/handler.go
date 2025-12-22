// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/market-making"
	"github.com/wyfcoding/financialTrading/internal/market-making/application"
	"github.com/wyfcoding/financialTrading/internal/market-making/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与做市相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedMarketMakingServiceServer
	app *application.MarketMakingService // 做市应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的做市应用服务
func NewGRPCHandler(app *application.MarketMakingService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// SetStrategy 设置做市策略
// 处理 gRPC SetStrategy 请求
func (h *GRPCHandler) SetStrategy(ctx context.Context, req *pb.SetStrategyRequest) (*pb.SetStrategyResponse, error) {
	// 调用应用服务设置策略
	id, err := h.app.SetStrategy(ctx, req.Symbol, req.Spread, req.MinOrderSize, req.MaxOrderSize, req.MaxPosition, req.Status)
	if err != nil {
		return nil, err
	}

	return &pb.SetStrategyResponse{
		StrategyId: id,
	}, nil
}

// GetStrategy 获取做市策略
func (h *GRPCHandler) GetStrategy(ctx context.Context, req *pb.GetStrategyRequest) (*pb.GetStrategyResponse, error) {
	strategy, err := h.app.GetStrategy(ctx, req.Symbol)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return &pb.GetStrategyResponse{}, nil
	}

	return &pb.GetStrategyResponse{
		Strategy: toProtoStrategy(strategy),
	}, nil
}

// GetPerformance 获取做市绩效
func (h *GRPCHandler) GetPerformance(ctx context.Context, req *pb.GetPerformanceRequest) (*pb.GetPerformanceResponse, error) {
	performance, err := h.app.GetPerformance(ctx, req.Symbol)
	if err != nil {
		return nil, err
	}
	if performance == nil {
		return &pb.GetPerformanceResponse{}, nil
	}

	return &pb.GetPerformanceResponse{
		Performance: toProtoPerformance(performance),
	}, nil
}

func toProtoStrategy(s *domain.QuoteStrategy) *pb.QuoteStrategy {
	return &pb.QuoteStrategy{
		Id:           s.ID,
		Symbol:       s.Symbol,
		Spread:       s.Spread,
		MinOrderSize: s.MinOrderSize,
		MaxOrderSize: s.MaxOrderSize,
		MaxPosition:  s.MaxPosition,
		Status:       string(s.Status),
		CreatedAt:    timestamppb.New(s.CreatedAt),
		UpdatedAt:    timestamppb.New(s.UpdatedAt),
	}
}

func toProtoPerformance(p *domain.MarketMakingPerformance) *pb.MarketMakingPerformance {
	return &pb.MarketMakingPerformance{
		Symbol:      p.Symbol,
		TotalPnl:    p.TotalPnL,
		TotalVolume: p.TotalVolume,
		TotalTrades: int32(p.TotalTrades),
		SharpeRatio: p.SharpeRatio,
		StartTime:   timestamppb.New(p.StartTime),
		EndTime:     timestamppb.New(p.EndTime),
	}
}
