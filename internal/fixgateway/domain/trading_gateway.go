// 生成摘要：从 gateway 合并到 fixgateway 域。交易网关属 FIX 网关子聚合。
// 关键实体：TradingSession（交易会话）、InboundOrder（入站订单）、GatewayMetrics。
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// 交易网关业务错误。
var (
	ErrGatewayUnavailable     = errors.New("trading gateway is unavailable")
	ErrTradingSessionNotFound = errors.New("trading session not found")
	ErrTradingSessionExpired  = errors.New("trading session has expired")
	ErrInvalidProtocol        = errors.New("unsupported trading protocol")
	ErrOrderRejectedByGW      = errors.New("order rejected by gateway")
	ErrGatewayRateLimit       = errors.New("rate limit exceeded")
	ErrGatewayMarketClosed    = errors.New("market is closed")
	ErrInstrumentNotTradable  = errors.New("instrument is not tradable")
)

// TradingProtocol 交易协议类型。
type TradingProtocol string

const (
	ProtocolFIX       TradingProtocol = "FIX"
	ProtocolREST      TradingProtocol = "REST"
	ProtocolGRPC      TradingProtocol = "GRPC"
	ProtocolWebSocket TradingProtocol = "WEBSOCKET"
	ProtocolBinary    TradingProtocol = "BINARY"
)

// InboundOrder 入站订单（网关接收的原始订单）。
type InboundOrder struct {
	ClientOrderID string            `json:"client_order_id"`
	AccountID     string            `json:"account_id"`
	Symbol        string            `json:"symbol"`
	Side          string            `json:"side"`
	Type          string            `json:"type"`
	Quantity      decimal.Decimal   `json:"quantity"`
	Price         decimal.Decimal   `json:"price"`
	StopPrice     decimal.Decimal   `json:"stop_price"`
	TimeInForce   string            `json:"time_in_force"`
	Protocol      TradingProtocol   `json:"protocol"`
	SessionID     string            `json:"session_id"`
	SourceIP      string            `json:"source_ip"`
	ReceivedAt    time.Time         `json:"received_at"`
	Metadata      map[string]string `json:"metadata"`
}

// TradingSession 交易会话。
type TradingSession struct {
	ID          string          `json:"id"`
	AccountID   string          `json:"account_id"`
	Protocol    TradingProtocol `json:"protocol"`
	SourceIP    string          `json:"source_ip"`
	Permissions []string        `json:"permissions"`
	RateLimit   int32           `json:"rate_limit"`
	CreatedAt   time.Time       `json:"created_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
	LastActive  time.Time       `json:"last_active"`
}

// IsExpired 会话是否过期。
func (s *TradingSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// HasPermission 是否具有指定权限。
func (s *TradingSession) HasPermission(perm string) bool {
	for _, p := range s.Permissions {
		if p == perm || p == "ALL" {
			return true
		}
	}
	return false
}

// GatewayMetrics 网关指标。
type GatewayMetrics struct {
	TotalOrders     int64           `json:"total_orders"`
	RejectedOrders  int64           `json:"rejected_orders"`
	AvgLatencyUs    int64           `json:"avg_latency_us"`
	P99LatencyUs    int64           `json:"p99_latency_us"`
	ActiveSessions  int32           `json:"active_sessions"`
	OrdersPerSecond decimal.Decimal `json:"orders_per_second"`
	Timestamp       time.Time       `json:"timestamp"`
}

// OrderValidator 订单校验器接口。
type OrderValidator interface {
	Validate(ctx context.Context, order *InboundOrder) (valid bool, reasons []string, err error)
}

// OrderRouter 订单路由器接口。
type OrderRouter interface {
	Route(ctx context.Context, order *InboundOrder) (targetVenue, targetService string, err error)
}

// SessionManager 会话管理器接口。
type SessionManager interface {
	CreateSession(ctx context.Context, accountID string, protocol TradingProtocol, sourceIP string) (*TradingSession, error)
	GetSession(ctx context.Context, sessionID string) (*TradingSession, error)
	RefreshSession(ctx context.Context, sessionID string) error
	CloseSession(ctx context.Context, sessionID string) error
}
