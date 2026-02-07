package application

import (
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/risk/domain"
)

// AssessRiskCommand 风险评估命令
type AssessRiskCommand struct {
	AssessmentID string
	UserID       string
	Symbol       string
	Side         string
	Quantity     float64
	Price        float64
}

// UpdateRiskLimitCommand 更新风险限额命令
type UpdateRiskLimitCommand struct {
	LimitID      string
	UserID       string
	LimitType    string
	LimitValue   float64
	CurrentValue float64
}

// TriggerCircuitBreakerCommand 触发熔断命令
type TriggerCircuitBreakerCommand struct {
	UserID         string
	TriggerReason  string
	AutoResetAfter int64 // 秒
}

// ResetCircuitBreakerCommand 重置熔断命令
type ResetCircuitBreakerCommand struct {
	UserID      string
	ResetReason string
}

// GenerateRiskAlertCommand 生成风险告警命令
type GenerateRiskAlertCommand struct {
	AlertID   string
	UserID    string
	AlertType string
	Severity  string
	Message   string
}

// UpdateRiskMetricsCommand 更新风险指标命令
type UpdateRiskMetricsCommand struct {
	UserID      string
	VaR95       float64
	VaR99       float64
	MaxDrawdown float64
	SharpeRatio float64
	Correlation float64
}

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
	UpdatedAt         int64  `json:"updated_at"`
}

// RiskMetricsDTO 风险指标 DTO
type RiskMetricsDTO struct {
	UserID      string `json:"user_id"`
	VaR95       string `json:"var_95"`
	VaR99       string `json:"var_99"`
	MaxDrawdown string `json:"max_drawdown"`
	SharpeRatio string `json:"sharpe_ratio"`
	Correlation string `json:"correlation"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// RiskLimitDTO 风险限额 DTO
type RiskLimitDTO struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	LimitType    string `json:"limit_type"`
	LimitValue   string `json:"limit_value"`
	CurrentValue string `json:"current_value"`
	IsExceeded   bool   `json:"is_exceeded"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// RiskAlertDTO 风险告警 DTO
type RiskAlertDTO struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	AlertType string `json:"alert_type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// CircuitBreakerDTO 风险熔断 DTO
type CircuitBreakerDTO struct {
	UserID        string `json:"user_id"`
	IsFired       bool   `json:"is_fired"`
	TriggerReason string `json:"trigger_reason"`
	FiredAt       int64  `json:"fired_at"`
	AutoResetAt   int64  `json:"auto_reset_at"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// CalculatePortfolioRiskRequest 组合风险计算请求 DTO
type CalculatePortfolioRiskRequest struct {
	Assets          []PortfolioAssetDTO `json:"assets"`
	CorrelationData [][]float64         `json:"correlation_data"`
	TimeHorizon     float64             `json:"time_horizon"`
	Simulations     int                 `json:"simulations"`
	ConfidenceLevel float64             `json:"confidence_level"`
}

type PortfolioAssetDTO struct {
	Symbol         string  `json:"symbol"`
	Position       string  `json:"position"`
	CurrentPrice   string  `json:"current_price"`
	Volatility     float64 `json:"volatility"`
	ExpectedReturn float64 `json:"expected_return"`
}

// CalculatePortfolioRiskResponse 组合风险计算响应 DTO
type CalculatePortfolioRiskResponse struct {
	TotalValue      string            `json:"total_value"`
	VaR             string            `json:"var"`
	ES              string            `json:"es"`
	ComponentVaR    map[string]string `json:"component_var"`
	Diversification string            `json:"diversification"`
}

func toRiskAssessmentDTO(a *domain.RiskAssessment) *RiskAssessmentDTO {
	if a == nil {
		return nil
	}
	return &RiskAssessmentDTO{
		AssessmentID:      a.ID,
		UserID:            a.UserID,
		Symbol:            a.Symbol,
		Side:              a.Side,
		Quantity:          a.Quantity.String(),
		Price:             a.Price.String(),
		RiskLevel:         string(a.RiskLevel),
		RiskScore:         a.RiskScore.String(),
		MarginRequirement: a.MarginRequirement.String(),
		IsAllowed:         a.IsAllowed,
		Reason:            a.Reason,
		CreatedAt:         a.CreatedAt.Unix(),
		UpdatedAt:         a.UpdatedAt.Unix(),
	}
}

func toRiskMetricsDTO(m *domain.RiskMetrics) *RiskMetricsDTO {
	if m == nil {
		return nil
	}
	return &RiskMetricsDTO{
		UserID:      m.UserID,
		VaR95:       m.VaR95.String(),
		VaR99:       m.VaR99.String(),
		MaxDrawdown: m.MaxDrawdown.String(),
		SharpeRatio: m.SharpeRatio.String(),
		Correlation: m.Correlation.String(),
		CreatedAt:   m.CreatedAt.Unix(),
		UpdatedAt:   m.UpdatedAt.Unix(),
	}
}

func toRiskLimitDTO(l *domain.RiskLimit) *RiskLimitDTO {
	if l == nil {
		return nil
	}
	return &RiskLimitDTO{
		ID:           l.ID,
		UserID:       l.UserID,
		LimitType:    l.LimitType,
		LimitValue:   l.LimitValue.String(),
		CurrentValue: l.CurrentValue.String(),
		IsExceeded:   l.IsExceeded,
		CreatedAt:    l.CreatedAt.Unix(),
		UpdatedAt:    l.UpdatedAt.Unix(),
	}
}

func toRiskAlertDTO(a *domain.RiskAlert) *RiskAlertDTO {
	if a == nil {
		return nil
	}
	return &RiskAlertDTO{
		ID:        a.ID,
		UserID:    a.UserID,
		AlertType: a.AlertType,
		Severity:  a.Severity,
		Message:   a.Message,
		CreatedAt: a.CreatedAt.Unix(),
		UpdatedAt: a.UpdatedAt.Unix(),
	}
}

func toRiskAlertDTOs(alerts []*domain.RiskAlert) []*RiskAlertDTO {
	result := make([]*RiskAlertDTO, 0, len(alerts))
	for _, a := range alerts {
		result = append(result, toRiskAlertDTO(a))
	}
	return result
}

func toCircuitBreakerDTO(cb *domain.CircuitBreaker) *CircuitBreakerDTO {
	if cb == nil {
		return nil
	}
	var firedAt int64
	var resetAt int64
	if cb.FiredAt != nil {
		firedAt = cb.FiredAt.Unix()
	}
	if cb.AutoResetAt != nil {
		resetAt = cb.AutoResetAt.Unix()
	}
	return &CircuitBreakerDTO{
		UserID:        cb.UserID,
		IsFired:       cb.IsFired,
		TriggerReason: cb.TriggerReason,
		FiredAt:       firedAt,
		AutoResetAt:   resetAt,
		CreatedAt:     cb.CreatedAt.Unix(),
		UpdatedAt:     cb.UpdatedAt.Unix(),
	}
}

func parseDecimal(value string) decimal.Decimal {
	dec, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero
	}
	return dec
}
