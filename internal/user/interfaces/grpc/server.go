package grpc

import (
	"context"
	"fmt"

	v1 "github.com/wyfcoding/financialtrading/go-api/user/v1"
	"github.com/wyfcoding/financialtrading/internal/user/application"
	"google.golang.org/grpc"
)

type Server struct {
	v1.UnimplementedUserServiceServer
	app *application.UserApplicationService
}

func NewServer(s *grpc.Server, app *application.UserApplicationService) *Server {
	srv := &Server{app: app}
	v1.RegisterUserServiceServer(s, srv)
	return srv
}

func (s *Server) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	var id uint
	fmt.Sscanf(req.Id, "%d", &id)
	u, err := s.app.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetUserResponse{User: &v1.User{Id: fmt.Sprint(u.ID), Email: u.Email, Name: u.Name, Phone: u.Phone, CreatedAt: u.CreatedAt.Unix()}}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	var id uint
	fmt.Sscanf(req.Id, "%d", &id)
	if err := s.app.UpdateUser(ctx, id, req.Name, req.Phone); err != nil {
		return nil, err
	}
	return &v1.UpdateUserResponse{Success: true}, nil
}
