package application

// SetStrategyCommand 设置做市策略命令
// 使用字符串保持与 API 兼容

type SetStrategyCommand struct {
	Symbol       string `json:"symbol"`
	Spread       string `json:"spread"`
	MinOrderSize string `json:"min_order_size"`
	MaxOrderSize string `json:"max_order_size"`
	MaxPosition  string `json:"max_position"`
	Status       string `json:"status"`
}

// StrategyDTO 做市策略 DTO

type StrategyDTO struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	Spread       string `json:"spread"`
	MinOrderSize string `json:"min_order_size"`
	MaxOrderSize string `json:"max_order_size"`
	MaxPosition  string `json:"max_position"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// PerformanceDTO 做市绩效 DTO

type PerformanceDTO struct {
	Symbol      string  `json:"symbol"`
	TotalPnL    float64 `json:"total_pnl"`
	TotalVolume float64 `json:"total_volume"`
	TotalTrades int32   `json:"total_trades"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time"`
}
