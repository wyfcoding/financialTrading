package application

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
	"github.com/wyfcoding/pkg/connectivity/fix"
)

// MessageProcessor 处理解析后的 FIX 报文并分发到内部服务
type MessageProcessor struct {
	sessionMgr *fix.SessionManager
	execClient domain.ExecutionClient
}

func NewMessageProcessor(sm *fix.SessionManager, ec domain.ExecutionClient) *MessageProcessor {
	return &MessageProcessor{
		sessionMgr: sm,
		execClient: ec,
	}
}

func (p *MessageProcessor) Process(ctx context.Context, sessionID string, msg *fix.Message) error {
	msgType := msg.Get(fix.TagMsgType)
	slog.Info("Processing FIX message", "session", sessionID, "type", msgType)

	switch msgType {
	case "D": // NewOrderSingle
		return p.handleNewOrder(ctx, sessionID, msg)
	case "V": // MarketDataRequest
		return p.handleMarketDataRequest(ctx, sessionID, msg)
	default:
		slog.Warn("Unknown FIX message type", "type", msgType)
	}
	return nil
}

func (p *MessageProcessor) handleNewOrder(ctx context.Context, sessionID string, msg *fix.Message) error {
	clOrdID := msg.Get(fix.TagClOrdID)
	symbol := msg.Get(fix.TagSymbol)
	qtyStr := msg.Get(fix.TagOrderQty)
	priceStr := msg.Get(fix.TagPrice)
	side := msg.Get(fix.TagSide) // 1=Buy, 2=Sell

	qty, _ := decimal.NewFromString(qtyStr)
	price, _ := decimal.NewFromString(priceStr)

	// 映射 Side
	internalSide := "BUY"
	if side == "2" {
		internalSide = "SELL"
	}

	slog.Info("FIX NewOrderReceived, routing to Execution", "clOrdID", clOrdID, "symbol", symbol)

	_, err := p.execClient.SubmitOrder(ctx, domain.FIXOrderCommand{
		ClOrdID:  clOrdID,
		UserID:   "INST_USER_001", // 模拟从 Session 映射
		Symbol:   symbol,
		Side:     internalSide,
		Price:    price,
		Quantity: qty,
	})
	return err
}

func (p *MessageProcessor) handleMarketDataRequest(ctx context.Context, sessionID string, msg *fix.Message) error {
	slog.Info("FIX MarketDataRequestReceived", "session", sessionID)
	return nil
}
