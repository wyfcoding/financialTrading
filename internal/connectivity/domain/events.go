package domain

import "time"

// FIXSessionConnectedEvent FIX 会话连接事件
type FIXSessionConnectedEvent struct {
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// FIXSessionDisconnectedEvent FIX 会话断开连接事件
type FIXSessionDisconnectedEvent struct {
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// FIXMessageReceivedEvent FIX 消息接收事件
type FIXMessageReceivedEvent struct {
	SessionID string    `json:"session_id"`
	MsgType   string    `json:"msg_type"`
	ClOrdID   string    `json:"cl_ord_id,omitempty"`
	Symbol    string    `json:"symbol,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// FIXOrderSubmittedEvent FIX 订单提交事件
type FIXOrderSubmittedEvent struct {
	ClOrdID   string    `json:"cl_ord_id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

// MarketDataUpdatedEvent 市场数据更新事件
type MarketDataUpdatedEvent struct {
	Symbol    string    `json:"symbol"`
	BidPrice  float64   `json:"bid_price"`
	AskPrice  float64   `json:"ask_price"`
	LastPrice float64   `json:"last_price"`
	Timestamp time.Time `json:"timestamp"`
}
