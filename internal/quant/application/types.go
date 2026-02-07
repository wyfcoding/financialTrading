package application

import "github.com/wyfcoding/financialtrading/internal/quant/domain"

// CreateStrategyCommand 创建策略命令
type CreateStrategyCommand struct {
	ID          string
	Name        string
	Description string
	Script      string
}

// UpdateStrategyCommand 更新策略命令
type UpdateStrategyCommand struct {
	ID          string
	Name        string
	Description string
	Status      string
	Script      string
}

// DeleteStrategyCommand 删除策略命令
type DeleteStrategyCommand struct {
	ID string
}

// RunBacktestCommand 运行回测命令
type RunBacktestCommand struct {
	BacktestID string
	StrategyID string
	Symbol     string
	StartTime  int64
	EndTime    int64
}

// GenerateSignalCommand 生成信号命令
type GenerateSignalCommand struct {
	SignalID   string
	StrategyID string
	Symbol     string
	Indicator  string
	Period     int
	Value      float64
	Confidence float64
}

// OptimizePortfolioCommand 优化组合命令
type OptimizePortfolioCommand struct {
	PortfolioID    string
	Symbols        []string
	ExpectedReturn float64
	RiskTolerance  float64
}

// AssessRiskCommand 风险评估命令
type AssessRiskCommand struct {
	AssessmentID string
	StrategyID   string
	Symbol       string
	Confidence   float64
}

// SignalDTO API/Query 输出结构
type SignalDTO struct {
	Symbol    string  `json:"symbol"`
	Indicator string  `json:"indicator"`
	Period    int     `json:"period"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

// ArbitrageOpportunityDTO 套利机会输出
type ArbitrageOpportunityDTO struct {
	Symbol      string `json:"symbol"`
	BuyVenue    string `json:"buy_venue"`
	SellVenue   string `json:"sell_venue"`
	Spread      string `json:"spread"`
	MaxQuantity int64  `json:"max_quantity"`
}

func toSignalDTO(signal *domain.Signal) *SignalDTO {
	if signal == nil {
		return nil
	}
	return &SignalDTO{
		Symbol:    signal.Symbol,
		Indicator: string(signal.Indicator),
		Period:    signal.Period,
		Value:     signal.Value,
		Timestamp: signal.Timestamp.Unix(),
	}
}
