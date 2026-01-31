package domain

import "time"

// StrategyCreatedEvent 做市策略创建事件
type StrategyCreatedEvent struct {
	StrategyID  string    `json:"strategy_id"`
	Symbol      string    `json:"symbol"`
	Spread      string    `json:"spread"`
	MinOrderSize string   `json:"min_order_size"`
	MaxOrderSize string   `json:"max_order_size"`
	MaxPosition  string   `json:"max_position"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

// StrategyUpdatedEvent 做市策略更新事件
type StrategyUpdatedEvent struct {
	StrategyID  string    `json:"strategy_id"`
	Symbol      string    `json:"symbol"`
	Spread      string    `json:"spread"`
	MinOrderSize string   `json:"min_order_size"`
	MaxOrderSize string   `json:"max_order_size"`
	MaxPosition  string   `json:"max_position"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

// StrategyActivatedEvent 做市策略激活事件
type StrategyActivatedEvent struct {
	StrategyID  string    `json:"strategy_id"`
	Symbol      string    `json:"symbol"`
	Timestamp   time.Time `json:"timestamp"`
}

// StrategyPausedEvent 做市策略暂停事件
type StrategyPausedEvent struct {
	StrategyID  string    `json:"strategy_id"`
	Symbol      string    `json:"symbol"`
	Timestamp   time.Time `json:"timestamp"`
}

// MarketMakingQuotePlacedEvent 做市报价下单事件
type MarketMakingQuotePlacedEvent struct {
	StrategyID  string    `json:"strategy_id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Price       string    `json:"price"`
	Quantity    string    `json:"quantity"`
	OrderID     string    `json:"order_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// MarketMakingPerformanceUpdatedEvent 做市性能更新事件
type MarketMakingPerformanceUpdatedEvent struct {
	Symbol      string    `json:"symbol"`
	TotalPnL    string    `json:"total_pnl"`
	TotalVolume string    `json:"total_volume"`
	TotalTrades int64     `json:"total_trades"`
	SharpeRatio string    `json:"sharpe_ratio"`
	Timestamp   time.Time `json:"timestamp"`
}
