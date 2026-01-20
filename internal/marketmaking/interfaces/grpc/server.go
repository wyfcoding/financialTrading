package grpc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	marketmakingv1 "github.com/wyfcoding/financialtrading/go-api/marketmaking/v1"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MarketMakingGrpcServer struct {
	marketmakingv1.UnimplementedMarketMakingServiceServer
	app *application.MarketMakingApplicationService
}

func NewMarketMakingGrpcServer(app *application.MarketMakingApplicationService) *MarketMakingGrpcServer {
	return &MarketMakingGrpcServer{app: app}
}

func (s *MarketMakingGrpcServer) SetStrategy(ctx context.Context, req *marketmakingv1.SetStrategyRequest) (*marketmakingv1.SetStrategyResponse, error) {
	cmd := application.SetStrategyCommand{
		Symbol:       req.Symbol,
		Spread:       fmt.Sprintf("%f", req.Spread),
		MinOrderSize: fmt.Sprintf("%f", req.MinOrderSize),
		MaxOrderSize: fmt.Sprintf("%f", req.MaxOrderSize),
		MaxPosition:  fmt.Sprintf("%f", req.MaxPosition),
		Status:       req.Status,
	}

	id, err := s.app.SetStrategy(ctx, cmd)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &marketmakingv1.SetStrategyResponse{StrategyId: id}, nil
}

func (s *MarketMakingGrpcServer) GetStrategy(ctx context.Context, req *marketmakingv1.GetStrategyRequest) (*marketmakingv1.GetStrategyResponse, error) {
	dto, err := s.app.GetStrategy(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dto == nil {
		return nil, status.Error(codes.NotFound, "strategy not found")
	}

	return &marketmakingv1.GetStrategyResponse{
		Strategy: &marketmakingv1.QuoteStrategy{
			Id:           dto.ID,
			Symbol:       dto.Symbol,
			Spread:       parseToFloat(dto.Spread),
			MinOrderSize: parseToFloat(dto.MinOrderSize),
			MaxOrderSize: parseToFloat(dto.MaxOrderSize),
			MaxPosition:  parseToFloat(dto.MaxPosition),
			Status:       dto.Status,
			CreatedAt:    timestamppb.New(time.UnixMilli(dto.CreatedAt)),
			UpdatedAt:    timestamppb.New(time.UnixMilli(dto.UpdatedAt)),
		},
	}, nil
}

func (s *MarketMakingGrpcServer) GetPerformance(ctx context.Context, req *marketmakingv1.GetPerformanceRequest) (*marketmakingv1.GetPerformanceResponse, error) {
	dto, err := s.app.GetPerformance(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &marketmakingv1.GetPerformanceResponse{
		Performance: &marketmakingv1.MarketMakingPerformance{
			Symbol:      dto.Symbol,
			TotalPnl:    dto.TotalPnL,
			TotalVolume: dto.TotalVolume,
			TotalTrades: dto.TotalTrades, // DTO is int32
			SharpeRatio: dto.SharpeRatio,
		},
	}, nil
}

func parseToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
