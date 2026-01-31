package application

import (
	"context"
	"log/slog"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
	"github.com/wyfcoding/pkg/connectivity/fix"
)

// ConnectivityCommandService 处理所有连接相关的写入操作（Commands）。
type ConnectivityCommandService struct {
	sessionMgr *fix.SessionManager
	execClient domain.ExecutionClient
	quotes     map[string]*Quote
	mu         sync.RWMutex
}

// Quote 行情快照
type Quote struct {
	Symbol    string
	BidPrice  float64
	AskPrice  float64
	LastPrice float64
}

// NewConnectivityCommandService 构造函数。
func NewConnectivityCommandService(sm *fix.SessionManager, ec domain.ExecutionClient) *ConnectivityCommandService {
	return &ConnectivityCommandService{
		sessionMgr: sm,
		execClient: ec,
		quotes:     make(map[string]*Quote),
	}
}

// ProcessMessage 处理 FIX 报文
func (s *ConnectivityCommandService) ProcessMessage(ctx context.Context, sessionID string, msg *fix.Message) error {
	msgType := msg.Get(fix.TagMsgType)
	slog.Info("Processing FIX message", "session", sessionID, "type", msgType)

	switch msgType {
	case "D": // NewOrderSingle
		return s.handleNewOrder(ctx, sessionID, msg)
	case "V": // MarketDataRequest
		return s.handleMarketDataRequest(ctx, sessionID, msg)
	default:
		slog.Warn("Unknown FIX message type", "type", msgType)
	}
	return nil
}

func (s *ConnectivityCommandService) handleNewOrder(ctx context.Context, sessionID string, msg *fix.Message) error {
	clOrdID := msg.Get(fix.TagClOrdID)
	symbol := msg.Get(fix.TagSymbol)
	qtyStr := msg.Get(fix.TagOrderQty)
	priceStr := msg.Get(fix.TagPrice)
	side := msg.Get(fix.TagSide)

	qty, _ := decimal.NewFromString(qtyStr)
	price, _ := decimal.NewFromString(priceStr)

	internalSide := "BUY"
	if side == "2" {
		internalSide = "SELL"
	}

	slog.Info("FIX NewOrderReceived, routing to Execution", "clOrdID", clOrdID, "symbol", symbol)

	_, err := s.execClient.SubmitOrder(ctx, domain.FIXOrderCommand{
		ClOrdID:  clOrdID,
		UserID:   "INST_USER_001",
		Symbol:   symbol,
		Side:     internalSide,
		Price:    price,
		Quantity: qty,
	})
	return err
}

func (s *ConnectivityCommandService) handleMarketDataRequest(ctx context.Context, sessionID string, msg *fix.Message) error {
	slog.Info("FIX MarketDataRequestReceived", "session", sessionID)
	return nil
}

// UpdateQuote 更新行情缓存
func (s *ConnectivityCommandService) UpdateQuote(symbol string, bid, ask, last float64) {
	s.mu.Lock()
	s.quotes[symbol] = &Quote{
		Symbol:    symbol,
		BidPrice:  bid,
		AskPrice:  ask,
		LastPrice: last,
	}
	s.mu.Unlock()
	slog.Debug("MarketGateway: Quote updated", "symbol", symbol, "last", last)
}

// GetQuote 获取行情快照
func (s *ConnectivityCommandService) GetQuote(symbol string) *Quote {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.quotes[symbol]
}
