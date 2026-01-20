package application

type SetStrategyCommand struct {
	Symbol       string
	Spread       string
	MinOrderSize string
	MaxOrderSize string
	MaxPosition  string
	Status       string
}

type StrategyDTO struct {
	ID           string
	Symbol       string
	Spread       string
	MinOrderSize string
	MaxOrderSize string
	MaxPosition  string
	Status       string
	CreatedAt    int64
	UpdatedAt    int64
}

type PerformanceDTO struct {
	Symbol      string
	TotalPnL    float64
	TotalVolume float64
	TotalTrades int32
	SharpeRatio float64
	StartTime   int64
	EndTime     int64
}
