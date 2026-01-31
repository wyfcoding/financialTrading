package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/pkg/connectivity/fix"
)

// ConnectivityQueryService 处理所有连接相关的查询操作（Queries）。
type ConnectivityQueryService struct {
	sessionMgr *fix.SessionManager
}

// NewConnectivityQueryService 构造函数。
func NewConnectivityQueryService(sm *fix.SessionManager) *ConnectivityQueryService {
	return &ConnectivityQueryService{sessionMgr: sm}
}

// GetSessionStatus 获取会话状态
func (s *ConnectivityQueryService) GetSessionStatus(ctx context.Context, sessionID string) (*fix.Session, error) {
	sess := s.sessionMgr.GetSession(sessionID)
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess, nil
}

// ListSessions 列出所有会话
func (s *ConnectivityQueryService) ListSessions(ctx context.Context) []*fix.Session {
	return s.sessionMgr.ListSessions()
}
