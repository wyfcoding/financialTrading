// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/quant/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与量化策略和回测相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedQuantServiceServer
	command *application.QuantCommandService
	query   *application.QuantQueryService
}

// NewHandler 创建 gRPC 处理器实例
// app: 注入的量化应用服务
func NewHandler(command *application.QuantCommandService, query *application.QuantQueryService) *Handler {
	return &Handler{command: command, query: query}
}

// GetSignal 获取信号 (Legacy)
func (h *Handler) GetSignal(ctx context.Context, req *pb.GetSignalRequest) (*pb.GetSignalResponse, error) {
	dto, err := h.query.GetSignal(ctx, req.Symbol, mapIndicator(req.Indicator), int(req.Period))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetSignalResponse{
		Signal: &pb.Signal{
			Symbol:    dto.Symbol,
			Indicator: req.Indicator,
			Value:     dto.Value,
			Period:    int32(dto.Period),
		},
	}, nil
}

// CreateStrategy 创建策略
func (h *Handler) CreateStrategy(ctx context.Context, req *pb.CreateStrategyRequest) (*pb.CreateStrategyResponse, error) {
	cmd := application.CreateStrategyCommand{
		Name:        req.Name,
		Description: req.Description,
		Script:      req.Script,
	}
	strategy, err := h.command.CreateStrategy(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create strategy: %v", err)
	}
	return &pb.CreateStrategyResponse{StrategyId: strategy.ID}, nil
}

// GetStrategy 获取策略
func (h *Handler) GetStrategy(ctx context.Context, req *pb.GetStrategyRequest) (*pb.GetStrategyResponse, error) {
	strategy, err := h.query.GetStrategy(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get strategy: %v", err)
	}
	if strategy == nil {
		return &pb.GetStrategyResponse{}, nil
	}
	return &pb.GetStrategyResponse{
		Strategy: toProtoStrategy(strategy),
	}, nil
}

// RunBacktest 运行回测
func (h *Handler) RunBacktest(ctx context.Context, req *pb.RunBacktestRequest) (*pb.RunBacktestResponse, error) {
	cmd := application.RunBacktestCommand{
		StrategyID: req.StrategyId,
		Symbol:     req.Symbol,
		StartTime:  req.StartTime.AsTime().UnixMilli(),
		EndTime:    req.EndTime.AsTime().UnixMilli(),
	}
	result, err := h.command.RunBacktest(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to run backtest: %v", err)
	}
	return &pb.RunBacktestResponse{BacktestId: result.ID}, nil
}

// GetBacktestResult 获取回测结果
func (h *Handler) GetBacktestResult(ctx context.Context, req *pb.GetBacktestResultRequest) (*pb.GetBacktestResultResponse, error) {
	result, err := h.query.GetBacktestResult(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get backtest result: %v", err)
	}
	if result == nil {
		return &pb.GetBacktestResultResponse{}, nil
	}
	return &pb.GetBacktestResultResponse{
		Result: toProtoBacktestResult(result),
	}, nil
}

func mapIndicator(i pb.IndicatorType) string {
	switch i {
	case pb.IndicatorType_RSI:
		return "RSI"
	case pb.IndicatorType_SMA:
		return "SMA"
	case pb.IndicatorType_EMA:
		return "EMA"
	default:
		return "UNKNOWN"
	}
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
	tr, _ := r.TotalReturn.Float64()
	md, _ := r.MaxDrawdown.Float64()
	sr, _ := r.SharpeRatio.Float64()

	return &pb.BacktestResult{
		Id:            r.ID,
		StrategyId:    r.StrategyID,
		Symbol:        r.Symbol,
		StartTime:     timestamppb.New(domain.MilliToTime(r.StartTime)),
		EndTime:       timestamppb.New(domain.MilliToTime(r.EndTime)),
		TotalReturn:   tr,
		MaxDrawdown:   md,
		SharpeRatio:   sr,
		TotalTrades:   int32(r.TotalTrades),
		WinningTrades: int32(r.WinningTrades),
		Status:        string(r.Status),
		CreatedAt:     timestamppb.New(r.CreatedAt),
	}
}

// OptimizePortfolio 优化投资组合
func (h *Handler) OptimizePortfolio(ctx context.Context, req *pb.OptimizePortfolioRequest) (*pb.OptimizePortfolioResponse, error) {
	cmd := application.OptimizePortfolioCommand{
		PortfolioID:    req.PortfolioId,
		Symbols:        req.Symbols,
		ExpectedReturn: req.ExpectedReturn,
		RiskTolerance:  req.RiskTolerance,
	}
	weights, err := h.command.OptimizePortfolio(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to optimize portfolio: %v", err)
	}
	return &pb.OptimizePortfolioResponse{
		PortfolioId: req.PortfolioId,
		Weights:     *weights,
	}, nil
}
