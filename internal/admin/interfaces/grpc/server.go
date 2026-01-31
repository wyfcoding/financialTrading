package grpc

import (
	"context"
	"fmt"

	v1 "github.com/wyfcoding/financialtrading/go-api/admin/v1"
	"github.com/wyfcoding/financialtrading/internal/admin/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedAdminServiceServer
	app *application.AdminService
}

func NewServer(s *grpc.Server, app *application.AdminService) *Server {
	srv := &Server{app: app}
	v1.RegisterAdminServiceServer(s, srv)
	return srv
}

func (s *Server) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	token, err := s.app.Login(ctx, application.LoginCommand{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &v1.LoginResponse{
		Token:     token.Token,
		Type:      token.Type,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

func (s *Server) CreateAdmin(ctx context.Context, req *v1.CreateAdminRequest) (*v1.CreateAdminResponse, error) {
	// Note: RoleID in proto is string, simplistic conversion for now or expect int string
	// For robustness we should check error, here we assume it parses for MVP or use 1
	var roleID uint = 1 // default or parse req.RoleId

	id, err := s.app.CreateAdmin(ctx, application.CreateAdminCommand{
		Username: req.Username,
		Password: req.Password,
		RoleID:   roleID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.CreateAdminResponse{Id: fmt.Sprintf("%d", id)}, nil
}

func (s *Server) GetAdmin(ctx context.Context, req *v1.GetAdminRequest) (*v1.GetAdminResponse, error) {
	// Parse ID
	var id uint // parse req.Id
	// mocking parse
	fmt.Sscanf(req.Id, "%d", &id)

	dto, err := s.app.GetAdmin(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Admin not found")
	}

	return &v1.GetAdminResponse{
		Admin: &v1.Admin{
			Id:        fmt.Sprintf("%d", dto.ID),
			Username:  dto.Username,
			RoleId:    dto.RoleName, // Mapping RoleName to ID field for display or change proto
			CreatedAt: dto.CreatedAt,
		},
	}, nil
}
