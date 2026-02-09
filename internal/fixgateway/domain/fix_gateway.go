// Package domain 提供了 FIX 协议网关的核心模型。
// 变更说明：实现 FIX (Financial Information eXchange) 协议网关基础逻辑，支持会话管理与标准消息模型。
package domain

import (
	"context"
	"time"
)

// FixSessionStatus FIX 会话状态
type FixSessionStatus string

const (
	FixSessionLogon   FixSessionStatus = "LOGON"
	FixSessionLogout  FixSessionStatus = "LOGOUT"
	FixSessionTimeout FixSessionStatus = "TIMEOUT"
)

// FixSession FIX 会话模型
type FixSession struct {
	SessionID     string
	CompID        string
	TargetID      string
	Version       string // e.g., FIX.4.4
	Status        FixSessionStatus
	LastMsgSeqIn  int
	LastMsgSeqOut int
	LastActiveAt  time.Time
}

// FixOrder 标准 FIX 订单消息提取
type FixOrder struct {
	ClOrdID      string // 客户端订单 ID
	Symbol       string
	Side         int // 1=Buy, 2=Sell
	OrdType      int // 1=Market, 2=Limit
	Price        float64
	OrderQty     float64
	TransactTime time.Time
}

// FixGatewayService FIX 网关服务接口
type FixGatewayService interface {
	Logon(ctx context.Context, compID, password string) (*FixSession, error)
	Logout(ctx context.Context, sessionID string) error
	SendExecutionReport(ctx context.Context, sessionID string, execution any) error
	Heartbeat(ctx context.Context, sessionID string) error
}

// FixMessageProcessor 消息处理器
type FixMessageProcessor struct {
	repo FixRepository
}

// FixRepository FIX 仓储接口
type FixRepository interface {
	GetSession(ctx context.Context, sessionID string) (*FixSession, error)
	SaveSession(ctx context.Context, session *FixSession) error
}

func NewFixMessageProcessor(repo FixRepository) *FixMessageProcessor {
	return &FixMessageProcessor{repo: repo}
}

// UpdateSequence 更新序列号并保活
func (p *FixMessageProcessor) UpdateSequence(ctx context.Context, sessionID string, seqIn int) error {
	session, err := p.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	session.LastMsgSeqIn = seqIn
	session.LastActiveAt = time.Now()
	return p.repo.SaveSession(ctx, session)
}
