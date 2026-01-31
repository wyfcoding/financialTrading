package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
	"github.com/wyfcoding/pkg/connectivity/fix"
)

// ConnectivityService 连接服务门面。
type ConnectivityService struct {
	Command *ConnectivityCommandService
	Query   *ConnectivityQueryService
}

// NewConnectivityService 构造函数。
func NewConnectivityService(sm *fix.SessionManager, ec domain.ExecutionClient, publisher domain.EventPublisher) *ConnectivityService {
	return &ConnectivityService{
		Command: NewConnectivityCommandService(sm, ec, publisher),
		Query:   NewConnectivityQueryService(sm),
	}
}

// --- Command Facade ---

func (s *ConnectivityService) ProcessMessage(ctx context.Context, sessionID string, msg *fix.Message) error {
	return s.Command.ProcessMessage(ctx, sessionID, msg)
}

func (s *ConnectivityService) UpdateQuote(symbol string, bid, ask, last float64) {
	s.Command.UpdateQuote(symbol, bid, ask, last)
}

func (s *ConnectivityService) GetQuote(symbol string) *Quote {
	return s.Command.GetQuote(symbol)
}

// --- Query Facade ---

func (s *ConnectivityService) GetSessionStatus(ctx context.Context, sessionID string) (*fix.Session, error) {
	return s.Query.GetSessionStatus(ctx, sessionID)
}

func (s *ConnectivityService) ListSessions(ctx context.Context) []*fix.Session {
	return s.Query.ListSessions(ctx)
}
