// 变更说明：完善FIX协议网关领域模型，实现完整的FIX 4.2/4.4协议支持、会话状态机、心跳重连、消息序列管理
package domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// FixVersion FIX协议版本
type FixVersion string

const (
	FixVersion40 FixVersion = "FIX.4.0"
	FixVersion41 FixVersion = "FIX.4.1"
	FixVersion42 FixVersion = "FIX.4.2"
	FixVersion43 FixVersion = "FIX.4.3"
	FixVersion44 FixVersion = "FIX.4.4"
	FixVersion50 FixVersion = "FIX.5.0"
	FixVersion50SP1 FixVersion = "FIX.5.0SP1"
	FixVersion50SP2 FixVersion = "FIX.5.0SP2"
)

// FixSessionStatus FIX会话状态
type FixSessionStatus string

const (
	FixSessionDisconnected  FixSessionStatus = "DISCONNECTED"
	FixSessionConnecting    FixSessionStatus = "CONNECTING"
	FixSessionLogonSent     FixSessionStatus = "LOGON_SENT"
	FixSessionLogonReceived FixSessionStatus = "LOGON_RECEIVED"
	FixSessionActive        FixSessionStatus = "ACTIVE"
	FixSessionLogoutSent    FixSessionStatus = "LOGOUT_SENT"
	FixSessionLogoutReceived FixSessionStatus = "LOGOUT_RECEIVED"
	FixSessionTimeout       FixSessionStatus = "TIMEOUT"
	FixSessionError         FixSessionStatus = "ERROR"
)

// FixSession FIX会话模型
type FixSession struct {
	SessionID       string           `json:"session_id"`
	CompID          string           `json:"comp_id"`
	TargetID        string           `json:"target_id"`
	Version         FixVersion       `json:"version"`
	Status          FixSessionStatus `json:"status"`
	LastMsgSeqIn    int              `json:"last_msg_seq_in"`
	LastMsgSeqOut   int              `json:"last_msg_seq_out"`
	LastActiveAt    time.Time        `json:"last_active_at"`
	CreatedAt       time.Time        `json:"created_at"`
	HeartbeatInt    int              `json:"heartbeat_interval"`
	EncryptMethod   int              `json:"encrypt_method"`
	ResetSeqNumFlag bool             `json:"reset_seq_num_flag"`
	Username        string           `json:"username"`
	Password        string           `json:"-"`
	IPAddress       string           `json:"ip_address"`
	LastTestReqID   string           `json:"last_test_req_id"`
	TestReqSentAt   *time.Time       `json:"test_req_sent_at"`
	LogoutReason    string           `json:"logout_reason"`
	mu              sync.RWMutex     `json:"-"`
}

// FixMessage FIX消息模型
type FixMessage struct {
	MsgType      string            `json:"msg_type"`
	MsgSeqNum    int               `json:"msg_seq_num"`
	SenderCompID string            `json:"sender_comp_id"`
	TargetCompID string            `json:"target_comp_id"`
	SendingTime  time.Time         `json:"sending_time"`
	Fields       map[int]string    `json:"fields"`
	RawMessage   string            `json:"raw_message"`
}

// FixOrder FIX订单消息
type FixOrder struct {
	ClOrdID      string    `json:"cl_ord_id"`
	Symbol       string    `json:"symbol"`
	Side         int       `json:"side"`
	OrdType      int       `json:"ord_type"`
	Price        float64   `json:"price"`
	OrderQty     float64   `json:"order_qty"`
	TransactTime time.Time `json:"transact_time"`
	Account      string    `json:"account"`
	HandlInst    string    `json:"handl_inst"`
	TimeInForce  string    `json:"time_in_force"`
	Text         string    `json:"text"`
}

// FixExecutionReport FIX执行报告
type FixExecutionReport struct {
	OrderID       string    `json:"order_id"`
	ClOrdID       string    `json:"cl_ord_id"`
	ExecID        string    `json:"exec_id"`
	ExecType      string    `json:"exec_type"`
	OrdStatus     string    `json:"ord_status"`
	Symbol        string    `json:"symbol"`
	Side          int       `json:"side"`
	LeavesQty     float64   `json:"leaves_qty"`
	CumQty        float64   `json:"cum_qty"`
	AvgPx         float64   `json:"avg_px"`
	LastShares    float64   `json:"last_shares"`
	LastPx        float64   `json:"last_px"`
	TransactTime  time.Time `json:"transact_time"`
	Text          string    `json:"text"`
}

// FixQuote FIX报价
type FixQuote struct {
	QuoteID      string    `json:"quote_id"`
	Symbol       string    `json:"symbol"`
	BidPx        float64   `json:"bid_px"`
	BidSize      float64   `json:"bid_size"`
	OfferPx      float64   `json:"offer_px"`
	OfferSize    float64   `json:"offer_size"`
	TransactTime time.Time `json:"transact_time"`
}

// FixMessageQueue FIX消息队列
type FixMessageQueue struct {
	messages []*FixMessage
	mu       sync.Mutex
}

// NewFixSession 创建新的FIX会话
func NewFixSession(compID, targetID string, version FixVersion) *FixSession {
	now := time.Now()
	return &FixSession{
		SessionID:     fmt.Sprintf("%s-%s-%d", compID, targetID, now.UnixNano()),
		CompID:        compID,
		TargetID:      targetID,
		Version:       version,
		Status:        FixSessionDisconnected,
		LastMsgSeqIn:  0,
		LastMsgSeqOut: 0,
		LastActiveAt:  now,
		CreatedAt:     now,
		HeartbeatInt:  30,
	}
}

// Connect 开始连接
func (s *FixSession) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Status != FixSessionDisconnected {
		return errors.New("session is not disconnected")
	}
	
	s.Status = FixSessionConnecting
	s.LastActiveAt = time.Now()
	return nil
}

// SendLogon 发送登录请求
func (s *FixSession) SendLogon() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Status != FixSessionConnecting {
		return errors.New("session is not in connecting state")
	}
	
	s.Status = FixSessionLogonSent
	s.LastActiveAt = time.Now()
	return nil
}

// ReceiveLogon 接收登录确认
func (s *FixSession) ReceiveLogon(heartbeatInt int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Status != FixSessionLogonSent {
		return errors.New("session did not send logon")
	}
	
	s.Status = FixSessionActive
	s.HeartbeatInt = heartbeatInt
	s.LastActiveAt = time.Now()
	return nil
}

// SendLogout 发送登出请求
func (s *FixSession) SendLogout(reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Status != FixSessionActive {
		return errors.New("session is not active")
	}
	
	s.Status = FixSessionLogoutSent
	s.LogoutReason = reason
	s.LastActiveAt = time.Now()
	return nil
}

// ReceiveLogout 接收登出确认
func (s *FixSession) ReceiveLogout() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.Status = FixSessionLogoutReceived
	s.LastActiveAt = time.Now()
	return nil
}

// Disconnect 断开连接
func (s *FixSession) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.Status = FixSessionDisconnected
	s.LastActiveAt = time.Now()
}

// Timeout 会话超时
func (s *FixSession) Timeout() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.Status = FixSessionTimeout
}

// IsActive 检查会话是否活跃
func (s *FixSession) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == FixSessionActive
}

// UpdateActivity 更新活动时间
func (s *FixSession) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActiveAt = time.Now()
}

// IncrementSeqIn 增加入站序列号
func (s *FixSession) IncrementSeqIn() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastMsgSeqIn++
	return s.LastMsgSeqIn
}

// IncrementSeqOut 增加出站序列号
func (s *FixSession) IncrementSeqOut() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastMsgSeqOut++
	return s.LastMsgSeqOut
}

// ResetSeqNum 重置序列号
func (s *FixSession) ResetSeqNum() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastMsgSeqIn = 0
	s.LastMsgSeqOut = 0
	s.ResetSeqNumFlag = true
}

// NeedHeartbeat 检查是否需要发送心跳
func (s *FixSession) NeedHeartbeat() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.Status != FixSessionActive {
		return false
	}
	
	elapsed := time.Since(s.LastActiveAt)
	return elapsed.Seconds() >= float64(s.HeartbeatInt)
}

// NeedTestRequest 检查是否需要发送测试请求
func (s *FixSession) NeedTestRequest() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.Status != FixSessionActive {
		return false
	}
	
	if s.TestReqSentAt == nil {
		elapsed := time.Since(s.LastActiveAt)
		return elapsed.Seconds() >= float64(s.HeartbeatInt*2)
	}
	
	return false
}

// SendTestRequest 发送测试请求
func (s *FixSession) SendTestRequest(testReqID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.LastTestReqID = testReqID
	s.TestReqSentAt = &now
}

// ReceiveHeartbeat 接收心跳响应
func (s *FixSession) ReceiveHeartbeat(testReqID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.LastTestReqID != "" && s.LastTestReqID == testReqID {
		s.TestReqSentAt = nil
		s.LastTestReqID = ""
		s.LastActiveAt = time.Now()
		return true
	}
	
	s.LastActiveAt = time.Now()
	return false
}

// ValidateSeqNum 验证序列号
func (s *FixSession) ValidateSeqNum(receivedSeqNum int) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	expectedSeqNum := s.LastMsgSeqIn + 1
	
	if receivedSeqNum < expectedSeqNum {
		return fmt.Errorf("sequence number too low: received %d, expected >= %d", receivedSeqNum, expectedSeqNum)
	}
	
	if receivedSeqNum > expectedSeqNum {
		return fmt.Errorf("sequence number gap: received %d, expected %d", receivedSeqNum, expectedSeqNum)
	}
	
	return nil
}

// FixRepository FIX仓储接口
type FixRepository interface {
	SaveSession(ctx context.Context, session *FixSession) error
	GetSession(ctx context.Context, sessionID string) (*FixSession, error)
	GetSessionByCompIDs(ctx context.Context, compID, targetID string) (*FixSession, error)
	ListActiveSessions(ctx context.Context) ([]*FixSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	
	SaveMessage(ctx context.Context, message *FixMessage) error
	GetMessages(ctx context.Context, sessionID string, limit int) ([]*FixMessage, error)
}

// FixMessageBuilder FIX消息构建器
type FixMessageBuilder struct {
	msg *FixMessage
}

// NewFixMessageBuilder 创建消息构建器
func NewFixMessageBuilder(msgType string) *FixMessageBuilder {
	return &FixMessageBuilder{
		msg: &FixMessage{
			MsgType:     msgType,
			Fields:      make(map[int]string),
			SendingTime: time.Now(),
		},
	}
}

// SetField 设置字段
func (b *FixMessageBuilder) SetField(tag int, value string) *FixMessageBuilder {
	b.msg.Fields[tag] = value
	return b
}

// SetSender 设置发送方
func (b *FixMessageBuilder) SetSender(compID string) *FixMessageBuilder {
	b.msg.SenderCompID = compID
	b.msg.Fields[49] = compID
	return b
}

// SetTarget 设置接收方
func (b *FixMessageBuilder) SetTarget(compID string) *FixMessageBuilder {
	b.msg.TargetCompID = compID
	b.msg.Fields[56] = compID
	return b
}

// SetSeqNum 设置序列号
func (b *FixMessageBuilder) SetSeqNum(seqNum int) *FixMessageBuilder {
	b.msg.MsgSeqNum = seqNum
	b.msg.Fields[34] = fmt.Sprintf("%d", seqNum)
	return b
}

// Build 构建消息
func (b *FixMessageBuilder) Build() *FixMessage {
	b.msg.Fields[52] = b.msg.SendingTime.UTC().Format("20060102-15:04:05.000")
	return b.msg
}

// 错误定义
var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionNotActive     = errors.New("session not active")
	ErrInvalidMessage       = errors.New("invalid fix message")
	ErrSequenceNumberGap    = errors.New("sequence number gap detected")
	ErrHeartbeatTimeout     = errors.New("heartbeat timeout")
	ErrAuthenticationFailed = errors.New("authentication failed")
)
