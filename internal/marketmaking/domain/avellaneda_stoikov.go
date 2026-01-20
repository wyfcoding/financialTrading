package domain

import (
	"math"

	"github.com/shopspring/decimal"
)

// AvellanedaStoikovModel 实现了经典的市场做市商模型
// 该模型旨在解决在存在存货风险的情况下，做市商的最优报价问题。
type AvellanedaStoikovModel struct {
	RiskAversion   float64 // Gamma (γ): 风险厌恶系数
	Volatility     float64 // Sigma (σ): 市场年化波动率
	OrderIntensity float64 // Kappa (κ): 订单簿深度/密度 (成交概率的衰减系数)
	TimeHorizon    float64 // T - t: 剩余时间 (通常设为 1.0 用于滚动窗口)
}

func NewAvellanedaStoikovModel(gamma, sigma, kappa float64) *AvellanedaStoikovModel {
	return &AvellanedaStoikovModel{
		RiskAversion:   gamma,
		Volatility:     sigma,
		OrderIntensity: kappa,
		TimeHorizon:    1.0,
	}
}

// QuoteParams 包含了模型计算出的报价参数
type QuoteParams struct {
	ReservationPrice decimal.Decimal
	Spread           decimal.Decimal
	BidPrice         decimal.Decimal
	AskPrice         decimal.Decimal
}

// CalculateQuotes 根据当前中间价和持仓量计算最优报价
// 公式 1: Reservation Price (r) = s - q * γ * σ^2 * (T - t)
// 公式 2: Optimal Spread (δ) = γ * σ^2 * (T - t) + (2/γ) * ln(1 + γ/κ)
func (m *AvellanedaStoikovModel) CalculateQuotes(
	midPrice decimal.Decimal,
	inventory decimal.Decimal, // q: 当前持仓量
) QuoteParams {
	s := midPrice.InexactFloat64()
	q := inventory.InexactFloat64()
	gamma := m.RiskAversion
	sigma := m.Volatility
	kappa := m.OrderIntensity
	t := m.TimeHorizon

	// 1. 计算保留价格 (Reservation Price)
	// 保留价格是做市商认为由于存货风险而导致的资产内部真实估值
	// 如果持仓多 (q > 0)，保留价格会低于中间价，诱导卖出；反之亦然。
	reservationPrice := s - q*gamma*math.Pow(sigma, 2)*t

	// 2. 计算最优价差 (Optimal Spread)
	// 价差由两部分组成：风险补偿部分 + 市场微观结构补偿部分
	spread := gamma*math.Pow(sigma, 2)*t + (2.0/gamma)*math.Log(1.0+(gamma/kappa))

	resDec := decimal.NewFromFloat(reservationPrice)
	spreadDec := decimal.NewFromFloat(spread)

	// 3. 计算最终报价
	bid := resDec.Sub(spreadDec.Div(decimal.NewFromFloat(2.0)))
	ask := resDec.Add(spreadDec.Div(decimal.NewFromFloat(2.0)))

	return QuoteParams{
		ReservationPrice: resDec,
		Spread:           spreadDec,
		BidPrice:         bid,
		AskPrice:         ask,
	}
}

// AvellanedaStoikovStrategy 包装了模型在做市策略中的应用
type AvellanedaStoikovStrategy struct {
	Model      *AvellanedaStoikovModel
	BaseConfig *QuoteStrategy
}

func (s *AvellanedaStoikovStrategy) GetQuotes(midPrice, inventory decimal.Decimal) (bid, ask decimal.Decimal) {
	params := s.Model.CalculateQuotes(midPrice, inventory)

	// 还可以加入基础策略的限制 (如最大/最小持仓限制)
	return params.BidPrice, params.AskPrice
}

// End of Avellaneda-Stoikov implementation
