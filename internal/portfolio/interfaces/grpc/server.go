package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/portfolio/v1"
	"github.com/wyfcoding/financialtrading/internal/portfolio/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedPortfolioServiceServer
	app *application.PortfolioAppService
}

func NewServer(app *application.PortfolioAppService) *Server {
	return &Server{app: app}
}

func (s *Server) GetPortfolio(ctx context.Context, req *pb.GetPortfolioRequest) (*pb.GetPortfolioResponse, error) {
	eq, upnl, rpnl, dpnl, err := s.app.GetPortfolio(ctx, req.UserId, req.Currency)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get portfolio: %v", err)
	}
	return &pb.GetPortfolioResponse{
		TotalEquity:   eq,
		UnrealizedPnl: upnl,
		RealizedPnl:   rpnl,
		DailyPnlPct:   dpnl,
		Currency:      req.Currency,
	}, nil
}

func (s *Server) GetPositions(ctx context.Context, req *pb.GetPositionsRequest) (*pb.GetPositionsResponse, error) {
	positions, err := s.app.GetPositions(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get positions: %v", err)
	}

	var pbPos []*pb.PositionItem
	for _, p := range positions {
		mktVal := p.Quantity * p.CurrentPrice
		pnl := (p.CurrentPrice - p.AvgPrice) * p.Quantity
		pct := 0.0
		if p.AvgPrice != 0 {
			pct = (p.CurrentPrice - p.AvgPrice) / p.AvgPrice
		}

		pbPos = append(pbPos, &pb.PositionItem{
			Symbol:        p.Symbol,
			Quantity:      p.Quantity,
			AvgPrice:      p.AvgPrice,
			CurrentPrice:  p.CurrentPrice,
			MarketValue:   mktVal,
			UnrealizedPnl: pnl,
			PnlPct:        pct,
			Type:          p.Type,
		})
	}
	return &pb.GetPositionsResponse{Positions: pbPos}, nil
}

func (s *Server) GetPerformance(ctx context.Context, req *pb.GetPerformanceRequest) (*pb.GetPerformanceResponse, error) {
	snaps, ret, sharpe, dd, err := s.app.GetPerformance(ctx, req.UserId, req.Timeframe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get performance: %v", err)
	}

	var points []*pb.PerformancePoint
	for _, snap := range snaps {
		val, _ := snap.TotalEquity.Float64()
		points = append(points, &pb.PerformancePoint{
			Timestamp: snap.Date.Format("2006-01-02"),
			Equity:    val,
		})
	}

	return &pb.GetPerformanceResponse{
		History:     points,
		TotalReturn: ret,
		SharpeRatio: sharpe,
		MaxDrawdown: dd,
	}, nil
}
