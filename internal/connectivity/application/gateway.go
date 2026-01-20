package application

import (
	"log/slog"
	"sync"

	"github.com/wyfcoding/pkg/connectivity/fix"
)

// MarketGateway 负责聚合多源行情并分发
type MarketGateway struct {
	sessionMgr *fix.SessionManager
	quotes     map[string]*Quote // Symbol -> Quote
	mu         sync.RWMutex
}

type Quote struct {
	Symbol    string
	BidPrice  float64
	AskPrice  float64
	LastPrice float64
}

func NewMarketGateway(sm *fix.SessionManager) *MarketGateway {
	return &MarketGateway{
		sessionMgr: sm,
		quotes:     make(map[string]*Quote),
	}
}

// UpdateQuote 接收来自 Kafka 或内部服务的更新并缓存
func (g *MarketGateway) UpdateQuote(symbol string, bid, ask, last float64) {
	g.mu.Lock()
	g.quotes[symbol] = &Quote{
		Symbol:    symbol,
		BidPrice:  bid,
		AskPrice:  ask,
		LastPrice: last,
	}
	g.mu.Unlock()

	// 实际场景应通过 SessionManager 广播 FIX MarketDataSnapshot (MsgType=W)
	slog.Debug("MarketGateway: Quote updated", "symbol", symbol, "last", last)
}

func (g *MarketGateway) GetQuote(symbol string) *Quote {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.quotes[symbol]
}
