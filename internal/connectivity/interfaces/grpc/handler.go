package grpc

import (
	"context"

	v1 "github.com/wyfcoding/financialtrading/go-api/connectivity/v1"
	"github.com/wyfcoding/financialtrading/internal/connectivity/application"
)

type Handler struct {
	v1.UnimplementedConnectivityServiceServer
	cmd   *application.ConnectivityCommandService
	query *application.ConnectivityQueryService
}

func NewHandler(cmd *application.ConnectivityCommandService, query *application.ConnectivityQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

func (h *Handler) GetSessionStatus(ctx context.Context, req *v1.GetSessionStatusRequest) (*v1.GetSessionStatusResponse, error) {
	sess, err := h.query.GetSessionStatus(ctx, req.SessionId)
	if err != nil {
		return nil, err
	}

	return &v1.GetSessionStatusResponse{
		SessionId:     sess.ID,
		Status:        "LOGGED_IN", // 演示简化
		SenderSeqNum:  sess.OutSeqNum,
		TargetSeqNum:  sess.InSeqNum,
		LastHeartbeat: sess.LastHeartbeat.Unix(),
	}, nil
}

func (h *Handler) ListActiveSessions(ctx context.Context, req *v1.ListActiveSessionsRequest) (*v1.ListActiveSessionsResponse, error) {
	sessions := h.query.ListSessions(ctx)
	var resp v1.ListActiveSessionsResponse
	for _, s := range sessions {
		resp.Sessions = append(resp.Sessions, &v1.GetSessionStatusResponse{
			SessionId:     s.ID,
			Status:        "LOGGED_IN",
			SenderSeqNum:  s.OutSeqNum,
			TargetSeqNum:  s.InSeqNum,
			LastHeartbeat: s.LastHeartbeat.Unix(),
		})
	}
	return &resp, nil
}
