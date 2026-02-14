package grpc

import (
	"context"
	"time"

	v1 "github.com/wyfcoding/financialtrading/go-api/fixgateway/v1"
	"github.com/wyfcoding/financialtrading/internal/fixgateway/application"
	"github.com/wyfcoding/financialtrading/internal/fixgateway/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedFixGatewayServiceServer
	app *application.FixApplicationService
}

func NewServer(app *application.FixApplicationService) *Server {
	return &Server{app: app}
}

func (s *Server) Logon(ctx context.Context, req *v1.LogonRequest) (*v1.LogonResponse, error) {
	session, err := s.app.Logon(ctx, req.CompId, req.TargetId, req.Password, req.FixVersion, 30)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "logon failed: %v", err)
	}

	return &v1.LogonResponse{
		SessionId: session.SessionID,
		Status:    string(session.Status),
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutResponse, error) {
	err := s.app.Logout(ctx, req.SessionId, "client_logout")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "logout failed: %v", err)
	}

	return &v1.LogoutResponse{Success: true}, nil
}

func (s *Server) SendOrder(ctx context.Context, req *v1.SendOrderRequest) (*v1.SendOrderResponse, error) {
	orderID, err := s.app.SendOrder(ctx, req.SessionId, domain.FixOrder{
		ClOrdID:      req.ClOrdId,
		Symbol:       req.Symbol,
		Side:         int(req.Side),
		OrdType:      int(req.OrdType),
		Price:        req.Price,
		OrderQty:     req.Quantity,
		TransactTime: time.Now(), // or from request if available
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send order failed: %v", err)
	}

	return &v1.SendOrderResponse{
		OrderId: orderID,
		Status:  "NEW", // Mock status
	}, nil
}

func (s *Server) GetSessionStatus(ctx context.Context, req *v1.GetSessionStatusRequest) (*v1.GetSessionStatusResponse, error) {
	session, err := s.app.GetSession(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	return &v1.GetSessionStatusResponse{
		Status: string(session.Status),
		SeqIn:  int32(session.LastMsgSeqIn),
		SeqOut: int32(session.LastMsgSeqOut),
	}, nil
}
