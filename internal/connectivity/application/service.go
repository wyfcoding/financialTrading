package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/pkg/connectivity/fix"
)

type ConnectivityService struct {
	sessionMgr *fix.SessionManager
}

func NewConnectivityService(sm *fix.SessionManager) *ConnectivityService {
	return &ConnectivityService{sessionMgr: sm}
}

func (s *ConnectivityService) GetSessionStatus(ctx context.Context, sessionID string) (*fix.Session, error) {
	sess := s.sessionMgr.GetSession(sessionID)
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess, nil
}

func (s *ConnectivityService) ListSessions(ctx context.Context) []*fix.Session {
	return s.sessionMgr.ListSessions()
}
