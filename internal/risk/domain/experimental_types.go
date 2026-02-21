//go:build risk_experimental
// +build risk_experimental

package domain

// Side 订单方向
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// Order 风险预检所需订单模型
type Order struct {
	OrderID   string  `json:"order_id"`
	Symbol    string  `json:"symbol"`
	Side      Side    `json:"side"`
	Quantity  float64 `json:"quantity"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

// TradeData What-if 输入模型
type TradeData struct {
	Symbol   string  `json:"symbol"`
	Side     Side    `json:"side"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// RiskDomainService 风险域服务占位定义（供级联限额扩展复用）
type RiskDomainService struct{}
