package grpc

import (
	"context"
	"fmt"

	v1 "github.com/wyfcoding/financialtrading/go-api/auth/v1"
	"github.com/wyfcoding/financialtrading/internal/auth/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedAuthServiceServer
	cmd   *application.AuthCommandService
	query *application.AuthQueryService
}

func NewHandler(cmd *application.AuthCommandService, query *application.AuthQueryService) *Server {
	return &Server{cmd: cmd, query: query}
}

func (s *Server) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	id, err := s.cmd.Register(ctx, application.RegisterCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	return &v1.RegisterResponse{UserId: fmt.Sprintf("%d", id)}, nil
}

func (s *Server) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	token, exp, err := s.cmd.Login(ctx, application.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &v1.LoginResponse{Token: token, Type: "Bearer", ExpiresAt: exp}, nil
}

func (s *Server) ValidateAPIKey(ctx context.Context, req *v1.ValidateAPIKeyRequest) (*v1.ValidateAPIKeyResponse, error) {
	ak, err := s.cmd.ValidateAPIKey(ctx, req.ApiKey)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid api key")
	}
	if ak == nil {
		return nil, status.Error(codes.NotFound, "api key not found")
	}
	return &v1.ValidateAPIKeyResponse{
		Secret:  "", // 我们不再向外暴露 Secret
		UserId:  ak.UserID,
		Enabled: ak.Enabled,
	}, nil
}

func (s *Server) VerifyAPIKey(ctx context.Context, req *v1.VerifyAPIKeyRequest) (*v1.VerifyAPIKeyResponse, error) {
	ak, err := s.cmd.VerifyAPIKey(ctx, req.ApiKey, req.Secret)
	if err != nil {
		return &v1.VerifyAPIKeyResponse{Valid: false}, nil
	}
	return &v1.VerifyAPIKeyResponse{
		Valid:  true,
		UserId: ak.UserID,
		Scopes: ak.Scopes,
	}, nil
}
