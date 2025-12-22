// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/notification"
	"github.com/wyfcoding/financialTrading/internal/notification/application"
	"github.com/wyfcoding/financialTrading/internal/notification/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与通知相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedNotificationServiceServer
	app *application.NotificationService // 通知应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的通知应用服务
func NewGRPCHandler(app *application.NotificationService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// SendNotification 发送通知
// 处理 gRPC SendNotification 请求
func (h *GRPCHandler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	// 调用应用服务发送通知
	id, err := h.app.SendNotification(ctx, req.UserId, req.Type, req.Subject, req.Content, req.Target)
	if err != nil {
		return nil, err
	}

	return &pb.SendNotificationResponse{
		NotificationId: id,
		Status:         "SENT", // 简化处理
	}, nil
}

// GetNotificationHistory 获取通知历史
func (h *GRPCHandler) GetNotificationHistory(ctx context.Context, req *pb.GetNotificationHistoryRequest) (*pb.GetNotificationHistoryResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	offset := 0 // 简单分页

	notifications, err := h.app.GetNotificationHistory(ctx, req.UserId, limit, offset)
	if err != nil {
		return nil, err
	}

	protoNotifications := make([]*pb.Notification, len(notifications))
	for i, n := range notifications {
		protoNotifications[i] = toProtoNotification(n)
	}

	return &pb.GetNotificationHistoryResponse{
		Notifications: protoNotifications,
	}, nil
}

func toProtoNotification(n *domain.Notification) *pb.Notification {
	return &pb.Notification{
		Id:           n.ID,
		UserId:       n.UserID,
		Type:         string(n.Type),
		Subject:      n.Subject,
		Content:      n.Content,
		Status:       string(n.Status),
		ErrorMessage: n.ErrorMessage,
		CreatedAt:    timestamppb.New(n.CreatedAt),
		SentAt:       timestamppb.New(n.SentAt),
	}
}
