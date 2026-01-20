package grpc

import (
	"context"
	"fmt"

	v1 "github.com/wyfcoding/financialtrading/go-api/notification/v1"
	"github.com/wyfcoding/financialtrading/internal/notification/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedNotificationServer
	app *application.NotificationManager
}

func NewServer(s *grpc.Server, app *application.NotificationManager) *Server {
	srv := &Server{app: app}
	v1.RegisterNotificationServer(s, srv)
	return srv
}

func (s *Server) SendNotification(ctx context.Context, req *v1.SendNotificationRequest) (*v1.SendNotificationResponse, error) {
	cmd := application.SendNotificationCommand{
		UserID:    req.UserId,
		Channel:   s.mapChannel(req.Channel),
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Content:   req.Content,
	}

	// For simplicity, we are sending synchronously. In prod, this would push to a queue or background job.
	id, err := s.app.SendNotification(ctx, cmd)
	if err != nil {
		// Log error but maybe returning success=false is better than gRPC error if we want to swallow failures
		// Here we return gRPC error if internal failure, or business failure
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.SendNotificationResponse{
		Success:        true,
		NotificationId: fmt.Sprint(id),
	}, nil
}

func (s *Server) GetHistory(ctx context.Context, req *v1.GetHistoryRequest) (*v1.GetHistoryResponse, error) {
	dtos, err := s.app.GetHistory(ctx, req.UserId, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &v1.GetHistoryResponse{
		Records: make([]*v1.NotificationRecord, len(dtos)),
	}

	for i, d := range dtos {
		resp.Records[i] = &v1.NotificationRecord{
			Id:        fmt.Sprint(d.ID),
			Channel:   s.mapChannelReverse(string(d.Channel)),
			Recipient: d.Recipient,
			Subject:   d.Subject,
			Content:   d.Content,
			Status:    string(d.Status),
			CreatedAt: timestamppb.New(d.CreatedAt),
		}
	}
	return resp, nil
}

func (s *Server) mapChannel(c v1.Channel) string {
	switch c {
	case v1.Channel_EMAIL:
		return "email"
	case v1.Channel_SMS:
		return "sms"
	case v1.Channel_PUSH:
		return "push"
	default:
		return "email" // default
	}
}

func (s *Server) mapChannelReverse(c string) v1.Channel {
	switch c {
	case "email":
		return v1.Channel_EMAIL
	case "sms":
		return v1.Channel_SMS
	case "push":
		return v1.Channel_PUSH
	default:
		return v1.Channel_CHANNEL_UNSPECIFIED
	}
}
