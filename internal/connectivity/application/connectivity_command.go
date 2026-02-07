package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
	"github.com/wyfcoding/pkg/connectivity/fix"
)

// ConnectivityCommandService 处理所有连接相关的写入操作（Commands）。
type ConnectivityCommandService struct {
	sessionMgr *fix.SessionManager
	execClient domain.ExecutionClient
	publisher  domain.EventPublisher
	quoteRepo  domain.QuoteRepository
}

// NewConnectivityCommandService 构造函数。
func NewConnectivityCommandService(
	sm *fix.SessionManager,
	ec domain.ExecutionClient,
	publisher domain.EventPublisher,
	quoteRepo domain.QuoteRepository,
) *ConnectivityCommandService {
	return &ConnectivityCommandService{
		sessionMgr: sm,
		execClient: ec,
		publisher:  publisher,
		quoteRepo:  quoteRepo,
	}
}

// ProcessMessage 处理 FIX 报文
func (s *ConnectivityCommandService) ProcessMessage(ctx context.Context, sessionID string, msg *fix.Message) error {
	msgType := msg.Get(fix.TagMsgType)
	slog.Info("Processing FIX message", "session", sessionID, "type", msgType)

	// 发布 FIX 消息接收事件
	clOrdID := msg.Get(fix.TagClOrdID)
	symbol := msg.Get(fix.TagSymbol)
	event := domain.FIXMessageReceivedEvent{
		SessionID: sessionID,
		MsgType:   msgType,
		ClOrdID:   clOrdID,
		Symbol:    symbol,
		Timestamp: time.Now(),
	}
	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, domain.FIXMessageReceivedEventType, sessionID, event)
	}

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

	// 发布 FIX 订单提交事件
	submitEvent := domain.FIXOrderSubmittedEvent{
		ClOrdID:   clOrdID,
		UserID:    "INST_USER_001",
		Symbol:    symbol,
		Side:      internalSide,
		Price:     price.String(),
		Quantity:  qty.String(),
		Timestamp: time.Now(),
	}
	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, domain.FIXOrderSubmittedEventType, clOrdID, submitEvent)
	}

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
	quote := &domain.Quote{
		Symbol:    symbol,
		BidPrice:  bid,
		AskPrice:  ask,
		LastPrice: last,
		UpdatedAt: time.Now(),
	}
	if s.quoteRepo != nil {
		if err := s.quoteRepo.Save(context.Background(), quote); err != nil {
			slog.Warn("failed to save quote to redis", "symbol", symbol, "error", err)
		}
	}

	// 发布市场数据更新事件
	event := domain.MarketDataUpdatedEvent{
		Symbol:    symbol,
		BidPrice:  bid,
		AskPrice:  ask,
		LastPrice: last,
		Timestamp: time.Now(),
	}
	if s.publisher != nil {
		_ = s.publisher.Publish(context.Background(), domain.MarketDataUpdatedEventType, symbol, event)
	}
}
