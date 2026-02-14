//go:build ignore
// +build ignore

package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/portfolio/v1"
	"github.com/wyfcoding/financialtrading/internal/portfolio/application"
	"github.com/wyfcoding/financialtrading/internal/portfolio/domain"
)

type PortfolioHandler struct {
	pb.UnimplementedPortfolioServiceServer
	app  *application.PortfolioService
	repo domain.PortfolioRepository
}

func NewPortfolioHandler(app *application.PortfolioService, repo domain.PortfolioRepository) *PortfolioHandler {
	return &PortfolioHandler{app: app, repo: repo}
}

func (h *PortfolioHandler) GetPortfolio(ctx context.Context, req *pb.GetPortfolioRequest) (*pb.GetPortfolioResponse, error) {
	equity, unrealized, realized, pct, err := h.app.GetOverview(ctx, req.UserId, req.Currency)
	if err != nil {
		return nil, err
	}

	return &pb.GetPortfolioResponse{
		TotalEquity:   equity,
		UnrealizedPnl: unrealized,
		RealizedPnl:   realized,
		DailyPnlPct:   pct,
		Currency:      req.Currency,
	}, nil
}

func (h *PortfolioHandler) GetPositions(ctx context.Context, req *pb.GetPositionsRequest) (*pb.GetPositionsResponse, error) {
	positions, err := h.app.GetPositions(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var res []*pb.PositionItem
	for _, p := range positions {
		res = append(res, &pb.PositionItem{
			Symbol:        p.Symbol,
			Quantity:      p.Qty,
			AvgPrice:      p.AvgPrice,
			CurrentPrice:  p.CurrentPrice,
			MarketValue:   p.MarketValue,
			UnrealizedPnl: p.UnrealizedPnL,
			PnlPct:        p.PnLPct,
			Type:          p.Type,
		})
	}
	return &pb.GetPositionsResponse{Positions: res}, nil
}

func (h *PortfolioHandler) GetPerformance(ctx context.Context, req *pb.GetPerformanceRequest) (*pb.GetPerformanceResponse, error) {
	history, perf, err := h.app.GetPerformance(ctx, req.UserId, req.Timeframe)
	if err != nil {
		return nil, err
	}

	var points []*pb.PerformancePoint
	for _, snap := range history {
		points = append(points, &pb.PerformancePoint{
			Timestamp: snap.Date.Format("2006-01-02"),
			Equity:    snap.TotalEquity.InexactFloat64(),
		})
	}

	return &pb.GetPerformanceResponse{
		History:     points,
		TotalReturn: perf.TotalReturn.InexactFloat64(),
		SharpeRatio: perf.SharpeRatio.InexactFloat64(),
		MaxDrawdown: perf.MaxDrawdown.InexactFloat64(),
	}, nil
}
