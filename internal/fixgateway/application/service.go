// 变更说明：完善FIX应用服务，增加心跳管理、消息处理、会话管理等完整功能
package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wyfcoding/financialtrading/internal/fixgateway/domain"
	"github.com/wyfcoding/pkg/messagequeue"
)

// FixApplicationService FIX应用服务
type FixApplicationService struct {
	repo      domain.FixRepository
	publisher messagequeue.EventPublisher
	logger    *slog.Logger
	sessions  sync.Map
}

// NewFixApplicationService 创建FIX应用服务
func NewFixApplicationService(
	repo domain.FixRepository,
	publisher messagequeue.EventPublisher,
	logger *slog.Logger,
) *FixApplicationService {
	return &FixApplicationService{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// Logon 处理登录请求
func (s *FixApplicationService) Logon(ctx context.Context, compID, targetID, password, version string, heartbeatInt int) (*domain.FixSession, error) {
	if password == "" {
		return nil, domain.ErrAuthenticationFailed
	}
	
	session := domain.NewFixSession(compID, targetID, domain.FixVersion(version))
	session.Password = password
	session.HeartbeatInt = heartbeatInt
	
	if err := session.Connect(); err != nil {
		return nil, err
	}
	
	if err := session.SendLogon(); err != nil {
		return nil, err
	}
	
	if err := session.ReceiveLogon(heartbeatInt); err != nil {
		return nil, err
	}
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}
	
	s.sessions.Store(session.SessionID, session)
	
	s.logger.InfoContext(ctx, "fix session logon success",
		"session_id", session.SessionID,
		"comp_id", compID,
		"target_id", targetID)
	
	return session, nil
}

// Logout 处理退出请求
func (s *FixApplicationService) Logout(ctx context.Context, sessionID, reason string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}
	
	if err := session.SendLogout(reason); err != nil {
		return err
	}
	
	session.ReceiveLogout()
	session.Disconnect()
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return err
	}
	
	s.sessions.Delete(sessionID)
	
	s.logger.InfoContext(ctx, "fix session logout success", "session_id", sessionID)
	return nil
}

// SendOrder 发送订单
func (s *FixApplicationService) SendOrder(ctx context.Context, sessionID string, order domain.FixOrder) (string, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session == nil {
		return "", domain.ErrSessionNotFound
	}
	
	if !session.IsActive() {
		return "", domain.ErrSessionNotActive
	}
	
	seqNum := session.IncrementSeqOut()
	
	msg := domain.NewFixMessageBuilder("D").
		SetSender(session.CompID).
		SetTarget(session.TargetID).
		SetSeqNum(seqNum).
		SetField(11, order.ClOrdID).
		SetField(55, order.Symbol).
		SetField(54, fmt.Sprintf("%d", order.Side)).
		SetField(40, fmt.Sprintf("%d", order.OrdType)).
		SetField(38, fmt.Sprintf("%.2f", order.OrderQty)).
		SetField(44, fmt.Sprintf("%.6f", order.Price)).
		SetField(1, order.Account).
		SetField(21, order.HandlInst).
		SetField(59, order.TimeInForce).
		Build()
	
	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		s.logger.ErrorContext(ctx, "failed to save order message", "error", err)
	}
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to save session", "error", err)
	}
	
	orderID := fmt.Sprintf("ORD-%d", time.Now().UnixNano())
	
	s.logger.InfoContext(ctx, "fix order sent",
		"session_id", sessionID,
		"order_id", orderID,
		"symbol", order.Symbol,
		"side", order.Side,
		"qty", order.OrderQty,
		"price", order.Price)
	
	return orderID, nil
}

// HandleExecutionReport 处理执行报告
func (s *FixApplicationService) HandleExecutionReport(ctx context.Context, sessionID string, report *domain.FixExecutionReport) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}
	
	session.IncrementSeqIn()
	session.UpdateActivity()
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to save session", "error", err)
	}
	
	s.publishEvent(ctx, "fix.execution_report", report.ClOrdID, report)
	
	s.logger.InfoContext(ctx, "execution report received",
		"session_id", sessionID,
		"order_id", report.OrderID,
		"exec_type", report.ExecType,
		"ord_status", report.OrdStatus)
	
	return nil
}

// HandleQuote 处理报价
func (s *FixApplicationService) HandleQuote(ctx context.Context, sessionID string, quote *domain.FixQuote) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}
	
	session.IncrementSeqIn()
	session.UpdateActivity()
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to save session", "error", err)
	}
	
	s.publishEvent(ctx, "fix.quote", quote.QuoteID, quote)
	
	s.logger.InfoContext(ctx, "quote received",
		"session_id", sessionID,
		"quote_id", quote.QuoteID,
		"symbol", quote.Symbol)
	
	return nil
}

// SendHeartbeat 发送心跳
func (s *FixApplicationService) SendHeartbeat(ctx context.Context, sessionID string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}
	
	if !session.IsActive() {
		return domain.ErrSessionNotActive
	}
	
	seqNum := session.IncrementSeqOut()
	
	msg := domain.NewFixMessageBuilder("0").
		SetSender(session.CompID).
		SetTarget(session.TargetID).
		SetSeqNum(seqNum).
		Build()
	
	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		s.logger.ErrorContext(ctx, "failed to save heartbeat message", "error", err)
	}
	
	session.UpdateActivity()
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to save session", "error", err)
	}
	
	return nil
}

// HandleHeartbeat 处理心跳响应
func (s *FixApplicationService) HandleHeartbeat(ctx context.Context, sessionID, testReqID string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}
	
	session.ReceiveHeartbeat(testReqID)
	session.IncrementSeqIn()
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to save session", "error", err)
	}
	
	return nil
}

// SendTestRequest 发送测试请求
func (s *FixApplicationService) SendTestRequest(ctx context.Context, sessionID string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}
	
	if !session.IsActive() {
		return domain.ErrSessionNotActive
	}
	
	testReqID := fmt.Sprintf("TEST-%d", time.Now().UnixNano())
	
	seqNum := session.IncrementSeqOut()
	
	msg := domain.NewFixMessageBuilder("1").
		SetSender(session.CompID).
		SetTarget(session.TargetID).
		SetSeqNum(seqNum).
		SetField(112, testReqID).
		Build()
	
	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		s.logger.ErrorContext(ctx, "failed to save test request message", "error", err)
	}
	
	session.SendTestRequest(testReqID)
	
	if err := s.repo.SaveSession(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, "failed to save session", "error", err)
	}
	
	return nil
}

// HeartbeatMonitor 心跳监控
func (s *FixApplicationService) HeartbeatMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sessions, err := s.repo.ListActiveSessions(ctx)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to list active sessions", "error", err)
				continue
			}
			
			for _, session := range sessions {
				if session.NeedHeartbeat() {
					_ = s.SendHeartbeat(ctx, session.SessionID)
				}
				
				if session.NeedTestRequest() {
					_ = s.SendTestRequest(ctx, session.SessionID)
				}
			}
		}
	}
}

// GetSession 获取会话
func (s *FixApplicationService) GetSession(ctx context.Context, sessionID string) (*domain.FixSession, error) {
	return s.repo.GetSession(ctx, sessionID)
}

// ListActiveSessions 列出活跃会话
func (s *FixApplicationService) ListActiveSessions(ctx context.Context) ([]*domain.FixSession, error) {
	return s.repo.ListActiveSessions(ctx)
}

// GetMessages 获取消息历史
func (s *FixApplicationService) GetMessages(ctx context.Context, sessionID string, limit int) ([]*domain.FixMessage, error) {
	return s.repo.GetMessages(ctx, sessionID, limit)
}

// publishEvent 发布事件
func (s *FixApplicationService) publishEvent(ctx context.Context, eventType, key string, event any) {
	if s.publisher == nil {
		return
	}
	if err := s.publisher.Publish(ctx, eventType, key, event); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish event", "event_type", eventType, "error", err)
	}
}
