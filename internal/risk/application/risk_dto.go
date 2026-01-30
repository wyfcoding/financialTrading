package application

// AssessRiskRequest 风险评估请求 DTO
type AssessRiskRequest struct {
	UserID   string `json:"user_id"`
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Quantity string `json:"quantity"`
	Price    string `json:"price"`
}

// RiskAssessmentDTO 风险评估 DTO
type RiskAssessmentDTO struct {
	AssessmentID      string `json:"assessment_id"`
	UserID            string `json:"user_id"`
	Symbol            string `json:"symbol"`
	Side              string `json:"side"`
	Quantity          string `json:"quantity"`
	Price             string `json:"price"`
	RiskLevel         string `json:"risk_level"`
	RiskScore         string `json:"risk_score"`
	MarginRequirement string `json:"margin_requirement"`
	IsAllowed         bool   `json:"is_allowed"`
	Reason            string `json:"reason"`
	CreatedAt         int64  `json:"created_at"`
}

// CalculatePortfolioRiskRequest 组合风险计算请求 DTO
type CalculatePortfolioRiskRequest struct {
	Assets          []PortfolioAssetDTO `json:"assets"`
	CorrelationData [][]float64         `json:"correlation_data"` // 相关系数矩阵
	TimeHorizon     float64             `json:"time_horizon"`     // 时间跨度(年)
	Simulations     int                 `json:"simulations"`      // 模拟次数
	ConfidenceLevel float64             `json:"confidence_level"` // 置信度
}

type PortfolioAssetDTO struct {
	Symbol         string  `json:"symbol"`
	Position       string  `json:"position"`        // 持仓数量
	CurrentPrice   string  `json:"current_price"`   // 当前价格
	Volatility     float64 `json:"volatility"`      // 年化波动率
	ExpectedReturn float64 `json:"expected_return"` // 预期年化收益率
}

// CalculatePortfolioRiskResponse 组合风险计算响应 DTO
type CalculatePortfolioRiskResponse struct {
	TotalValue      string            `json:"total_value"`
	VaR             string            `json:"var"`
	ES              string            `json:"es"`
	ComponentVaR    map[string]string `json:"component_var"`
	Diversification string            `json:"diversification"`
}
