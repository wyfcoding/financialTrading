package domain

import (
	"context"
	"time"
)

type SessionStatus string

const (
	SessionLogon    SessionStatus = "LOGON"
	SessionLogout   SessionStatus = "LOGOUT"
	SessionError    SessionStatus = "ERROR"
)

type FIXSession struct {
	ID           string
	TargetCompID string
	SenderCompID string
	Status       SessionStatus
	LastMsgTime  time.Time
}

type TradeGateway interface {
	SendOrder(ctx context.Context, order interface{}) error
	OnExecutionReport(handler func(report interface{}))
}
