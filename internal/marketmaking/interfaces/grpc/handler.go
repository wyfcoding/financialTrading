// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/goapi/marketmaking/v1"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/application"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与做市相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedMarketMakingServiceServer
	app *application.MarketMakingService // 做市应用服务
}

// NewHandler 创建 gRPC 处理器实例
// app: 注入的做市应用服务
func NewHandler(app *application.MarketMakingService) *Handler {
	return &Handler{
		app: app,
	}
}

// SetStrategy 设置做市策略
// 处理 gRPC SetStrategy 请求
func (h *Handler) SetStrategy(ctx context.Context, req *pb.SetStrategyRequest) (*pb.SetStrategyResponse, error) {
	start := time.Now()
	slog.Info("gRPC SetStrategy received", "symbol", req.Symbol, "spread", req.Spread, "status", req.Status)

	// 调用应用服务设置策略
	id, err := h.app.SetStrategy(ctx, req.Symbol,
		decimal.NewFromFloat(req.Spread),
		decimal.NewFromFloat(req.MinOrderSize),
		decimal.NewFromFloat(req.MaxOrderSize),
		decimal.NewFromFloat(req.MaxPosition),
		req.Status)
	if err != nil {
		slog.Error("gRPC SetStrategy failed", "symbol", req.Symbol, "error", err, "duration", time.Since(start))
		return nil, err
	}

	slog.Info("gRPC SetStrategy successful", "symbol", req.Symbol, "strategy_id", id, "duration", time.Since(start))
	return &pb.SetStrategyResponse{
		StrategyId: id,
	}, nil
}

// GetStrategy 获取做市策略
func (h *Handler) GetStrategy(ctx context.Context, req *pb.GetStrategyRequest) (*pb.GetStrategyResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetStrategy received", "symbol", req.Symbol)

	strategy, err := h.app.GetStrategy(ctx, req.Symbol)
	if err != nil {
		slog.Error("gRPC GetStrategy failed", "symbol", req.Symbol, "error", err, "duration", time.Since(start))
		return nil, err
	}
	if strategy == nil {
		slog.Debug("gRPC GetStrategy successful (not found)", "symbol", req.Symbol, "duration", time.Since(start))
		return &pb.GetStrategyResponse{}, nil
	}

	slog.Debug("gRPC GetStrategy successful", "symbol", req.Symbol, "duration", time.Since(start))
	return &pb.GetStrategyResponse{
		Strategy: toProtoStrategy(strategy),
	}, nil
}

// GetPerformance 获取做市绩效
func (h *Handler) GetPerformance(ctx context.Context, req *pb.GetPerformanceRequest) (*pb.GetPerformanceResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetPerformance received", "symbol", req.Symbol)

	performance, err := h.app.GetPerformance(ctx, req.Symbol)
	if err != nil {
		slog.Error("gRPC GetPerformance failed", "symbol", req.Symbol, "error", err, "duration", time.Since(start))
		return nil, err
	}
	if performance == nil {
		slog.Debug("gRPC GetPerformance successful (not found)", "symbol", req.Symbol, "duration", time.Since(start))
		return &pb.GetPerformanceResponse{}, nil
	}

	slog.Debug("gRPC GetPerformance successful", "symbol", req.Symbol, "duration", time.Since(start))
	return &pb.GetPerformanceResponse{
		Performance: toProtoPerformance(performance),
	}, nil
}

func toProtoStrategy(s *domain.QuoteStrategy) *pb.QuoteStrategy {
	return &pb.QuoteStrategy{
		Id:           s.Symbol,
		Symbol:       s.Symbol,
		Spread:       s.Spread.InexactFloat64(),
		MinOrderSize: s.MinOrderSize.InexactFloat64(),
		MaxOrderSize: s.MaxOrderSize.InexactFloat64(),
		MaxPosition:  s.MaxPosition.InexactFloat64(),
		Status:       string(s.Status),
		CreatedAt:    timestamppb.New(s.CreatedAt),
		UpdatedAt:    timestamppb.New(s.UpdatedAt),
	}
}

func toProtoPerformance(p *domain.MarketMakingPerformance) *pb.MarketMakingPerformance {
	return &pb.MarketMakingPerformance{
		Symbol:      p.Symbol,
		TotalPnl:    p.TotalPnL.InexactFloat64(),
		TotalVolume: p.TotalVolume.InexactFloat64(),
		TotalTrades: int32(p.TotalTrades),
		SharpeRatio: p.SharpeRatio.InexactFloat64(),
		StartTime:   timestamppb.New(p.StartTime),
		EndTime:     timestamppb.New(p.EndTime),
	}
}
