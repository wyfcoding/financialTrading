package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/marginlending/v1"
	"github.com/wyfcoding/financialtrading/internal/marginlending/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedMarginLendingServiceServer
	app *application.MarginAppService
}

func NewServer(app *application.MarginAppService) *Server {
	return &Server{app: app}
}

func (s *Server) EvaluateMargin(ctx context.Context, req *pb.EvaluateMarginRequest) (*pb.EvaluateMarginResponse, error) {
	eligible, reqMargin, leverage, err := s.app.EvaluateMargin(ctx, req.UserId, req.Symbol, req.Quantity, req.Price)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluate failed: %v", err)
	}
	return &pb.EvaluateMarginResponse{
		Eligible:          eligible,
		RequiredMargin:    reqMargin,
		AvailableLeverage: leverage,
	}, nil
}

func (s *Server) LockCollateral(ctx context.Context, req *pb.LockCollateralRequest) (*pb.LockCollateralResponse, error) {
	lockID, success, err := s.app.LockCollateral(ctx, req.UserId, req.Asset, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lock failed: %v", err)
	}
	return &pb.LockCollateralResponse{
		LockId:  lockID,
		Success: success,
	}, nil
}

func (s *Server) MarginCall(ctx context.Context, req *pb.MarginCallRequest) (*pb.MarginCallResponse, error) {
	mm, eq, liq, err := s.app.MarginCall(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "margin call check failed: %v", err)
	}
	return &pb.MarginCallResponse{
		MaintenanceMargin: mm,
		CurrentEquity:     eq,
		IsLiquidatable:    liq,
	}, nil
}
