package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/risk/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedRiskServiceServer
	app *application.RiskApplicationService
}

func NewServer(s *grpc.Server, app *application.RiskApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterRiskServiceServer(s, srv)
	return srv
}

func (s *Server) CheckRisk(ctx context.Context, req *v1.CheckRiskRequest) (*v1.CheckRiskResponse, error) {
	passed, reason := s.app.CheckRisk(ctx, req.UserId, req.Symbol, req.Quantity, req.Price)
	return &v1.CheckRiskResponse{
		Passed: passed,
		Reason: reason,
	}, nil
}

func (s *Server) SetRiskLimit(ctx context.Context, req *v1.SetRiskLimitRequest) (*v1.SetRiskLimitResponse, error) {
	err := s.app.SetRiskLimit(ctx, req.UserId, req.MaxOrderSize, req.MaxDailyLoss)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.SetRiskLimitResponse{Success: true}, nil
}
