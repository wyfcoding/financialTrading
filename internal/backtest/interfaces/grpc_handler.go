package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/backtest/v1"
	"github.com/wyfcoding/financialtrading/internal/backtest/application"
)

type BacktestHandler struct {
	pb.UnimplementedBacktestServiceServer
	appService *application.BacktestApplicationService
}

func NewBacktestHandler(appService *application.BacktestApplicationService) *BacktestHandler {
	return &BacktestHandler{
		appService: appService,
	}
}

func (h *BacktestHandler) RunBacktest(ctx context.Context, req *pb.RunBacktestRequest) (*pb.RunBacktestResponse, error) {
	cmd := application.RunBacktestCommand{
		StrategyID:     req.StrategyId,
		Symbol:         req.Symbol,
		StartTime:      req.StartTime.AsTime(),
		EndTime:        req.EndTime.AsTime(),
		InitialCapital: req.InitialCapital,
	}

	taskID, err := h.appService.RunBacktest(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pb.RunBacktestResponse{
		TaskId: taskID,
		Status: "ACCEPTED",
	}, nil
}

func (h *BacktestHandler) GetBacktestReport(ctx context.Context, req *pb.GetBacktestReportRequest) (*pb.GetBacktestReportResponse, error) {
	report, err := h.appService.GetReport(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	return &pb.GetBacktestReportResponse{
		TaskId:      report.TaskID,
		TotalReturn: report.TotalReturn,
		SharpeRatio: report.SharpeRatio,
		MaxDrawdown: report.MaxDrawdown,
		TotalTrades: int32(report.TotalTrades),
		WinRate:     report.WinRate,
	}, nil
}
