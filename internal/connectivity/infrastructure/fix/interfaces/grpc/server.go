package grpc

import (
	"context"
	"strconv"
	"time"

	v1 "github.com/wyfcoding/financialtrading/go-api/fixgateway/v1"
	"github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/fix/application"
	"github.com/wyfcoding/financialtrading/internal/connectivity/infrastructure/fix/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedFixGatewayServiceServer
	app *application.FixApplicationService
}

func NewServer(app *application.FixApplicationService) *Server {
	return &Server{app: app}
}

func (s *Server) Logon(ctx context.Context, req *v1.LogonRequest) (*v1.LogonResponse, error) {
	heartbeat := int(req.GetHeartBtInt())
	if heartbeat <= 0 {
		heartbeat = 30
	}

	session, err := s.app.Logon(
		ctx,
		req.GetSenderCompId(),
		req.GetTargetCompId(),
		req.GetPassword(),
		req.GetFixVersion(),
		heartbeat,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "logon failed: %v", err)
	}

	return &v1.LogonResponse{
		SessionId:    session.SessionID,
		Status:       toProtoSessionState(session.Status),
		HeartBtInt:   int32(session.HeartbeatInt),
		ServerCompId: session.TargetID,
		LogonTime:    timestamppb.New(time.Now()),
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
	price, err := parseDecimalField("price", req.GetPrice())
	if err != nil {
		return nil, err
	}

	quantityRaw := req.GetQuantity()
	if quantityRaw == "" {
		quantityRaw = req.GetOrderQty()
	}
	quantity, err := parseDecimalField("quantity", quantityRaw)
	if err != nil {
		return nil, err
	}

	orderID, err := s.app.SendOrder(ctx, req.GetSessionId(), domain.FixOrder{
		ClOrdID:      req.GetClOrdId(),
		Symbol:       req.GetSymbol(),
		Side:         int(req.GetSide()),
		OrdType:      int(req.GetOrdType()),
		Price:        price,
		OrderQty:     quantity,
		TransactTime: time.Now(),
		Account:      req.GetAccount(),
		TimeInForce:  toFixTimeInForce(req.GetTimeInForce()),
		Text:         req.GetText(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send order failed: %v", err)
	}

	return &v1.SendOrderResponse{
		ClOrdId: req.GetClOrdId(),
		OrderId: orderID,
		Status:  v1.OrdStatus_ORD_STATUS_NEW,
		Text:    "accepted",
	}, nil
}

func (s *Server) GetSessionStatus(ctx context.Context, req *v1.GetSessionStatusRequest) (*v1.SessionStatus, error) {
	session, err := s.app.GetSession(ctx, req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	return &v1.SessionStatus{
		SessionId:        session.SessionID,
		SenderCompId:     session.CompID,
		TargetCompId:     session.TargetID,
		State:            toProtoSessionState(session.Status),
		IncomingSeqNum:   int32(session.LastMsgSeqIn),
		OutgoingSeqNum:   int32(session.LastMsgSeqOut),
		LogonTime:        timestamppb.New(session.CreatedAt),
		LastHeartbeat:    timestamppb.New(session.LastActiveAt),
		MessagesSent:     int32(session.LastMsgSeqOut),
		MessagesReceived: int32(session.LastMsgSeqIn),
	}, nil
}

func parseDecimalField(name, value string) (float64, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid %s: %q", name, value)
	}
	return parsed, nil
}

func toProtoSessionState(state domain.FixSessionStatus) v1.SessionState {
	switch state {
	case domain.FixSessionDisconnected:
		return v1.SessionState_SESSION_STATE_DISCONNECTED
	case domain.FixSessionConnecting:
		return v1.SessionState_SESSION_STATE_CONNECTING
	case domain.FixSessionLogonSent, domain.FixSessionLogonReceived:
		return v1.SessionState_SESSION_STATE_LOGGING_ON
	case domain.FixSessionActive:
		return v1.SessionState_SESSION_STATE_ACTIVE
	case domain.FixSessionLogoutSent, domain.FixSessionLogoutReceived:
		return v1.SessionState_SESSION_STATE_LOGGING_OUT
	case domain.FixSessionTimeout:
		return v1.SessionState_SESSION_STATE_RECONNECTING
	case domain.FixSessionError:
		return v1.SessionState_SESSION_STATE_ERROR
	default:
		return v1.SessionState_SESSION_STATE_UNSPECIFIED
	}
}

func toFixTimeInForce(tif v1.TimeInForce) string {
	switch tif {
	case v1.TimeInForce_TIME_IN_FORCE_DAY:
		return "0"
	case v1.TimeInForce_TIME_IN_FORCE_GOOD_TILL_CANCEL:
		return "1"
	case v1.TimeInForce_TIME_IN_FORCE_AT_THE_OPENING:
		return "2"
	case v1.TimeInForce_TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		return "3"
	case v1.TimeInForce_TIME_IN_FORCE_FILL_OR_KILL:
		return "4"
	case v1.TimeInForce_TIME_IN_FORCE_GOOD_TILL_CROSSING:
		return "5"
	case v1.TimeInForce_TIME_IN_FORCE_GOOD_TILL_DATE:
		return "6"
	case v1.TimeInForce_TIME_IN_FORCE_AT_THE_CLOSE:
		return "7"
	default:
		return ""
	}
}
