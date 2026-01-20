package grpc

import (
	"context"
	"fmt"

	v1 "github.com/wyfcoding/financialtrading/go-api/auth/v1"
	"github.com/wyfcoding/financialtrading/internal/auth/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedAuthServiceServer
	app *application.AuthApplicationService
}

func NewServer(s *grpc.Server, app *application.AuthApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterAuthServiceServer(s, srv)
	return srv
}

func (s *Server) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	id, err := s.app.Register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	return &v1.RegisterResponse{UserId: fmt.Sprintf("%d", id)}, nil
}

func (s *Server) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	token, exp, err := s.app.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &v1.LoginResponse{Token: token, Type: "Bearer", ExpiresAt: exp}, nil
}
