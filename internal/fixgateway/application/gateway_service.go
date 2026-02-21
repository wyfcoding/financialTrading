package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/fixgateway/domain"
	"github.com/wyfcoding/pkg/connectivity/fix"
)

// 生成摘要：FIX 网关应用逻辑实现。
// 关键改动：桥接 FIX 消息层与领域层。

type FixAppService struct {
	repo    domain.Repository
	manager *fix.SessionManager
}

func NewFixAppService(repo domain.Repository, manager *fix.SessionManager) *FixAppService {
	return &FixAppService{
		repo:    repo,
		manager: manager,
	}
}

// HandleIncomingMessage 处理从交易所收到的消息。
func (s *FixAppService) HandleIncomingMessage(ctx context.Context, sessionID string, msg *fix.Message) error {
	sess, err := s.repo.FindByID(sessionID)
	if err != nil {
		slog.Error("session not found", "sessionID", sessionID)
		return err
	}

	// 1. 更新接收序列号
	if seq, seqErr := msg.GetInt(fix.TagMsgSeqNum); seqErr == nil {
		if int64(seq) > sess.InSeqNum {
			sess.InSeqNum = int64(seq)
		}
	}

	// 2. 根据消息类型处理
	msgType := msg.Get(fix.TagMsgType)
	switch msgType {
	case fix.MsgTypeExecutionReport:
		slog.Info("received execution report", "sessionID", sessionID, "orderID", msg.Get(37))
	case fix.MsgTypeHeartbeat:
		sess.LastHeartbeat = time.Now()
	}

	// 3. 持久化
	return s.repo.Save(sess)
}

// SendOrder 发送新订单。
func (s *FixAppService) SendOrder(ctx context.Context, sessionID string, orderMsg *fix.Message) error {
	session := s.manager.GetSession(sessionID)
	if session == nil {
		return fix.ErrSessionNotFound
	}

	// 发送消息
	if err := session.SendMessage(orderMsg); err != nil {
		return err
	}

	// 同步增加发送序列号（简化处理）
	sess, err := s.repo.FindByID(sessionID)
	if err != nil {
		return err
	}
	sess.OutSeqNum++
	return s.repo.Save(sess)
}
