package domain

import (
	"github.com/wyfcoding/pkg/algorithm/finance"
)

// AmericanOptionParams 定义美国期权合约参数
type AmericanOptionParams struct {
	S0    float64
	K     float64
	T     float64
	R     float64
	Sigma float64
	IsPut bool
	Paths int
	Steps int
}

// LSMPricer 实现了 Longstaff-Schwartz (LSM) 算法
type LSMPricer struct {
	impl *finance.LSMPricer
}

func NewLSMPricer() *LSMPricer {
	return &LSMPricer{
		impl: finance.NewLSMPricer(2),
	}
}

// Price 计算美国期权的当前公允价值
func (p *LSMPricer) Price(params AmericanOptionParams) (float64, error) {
	pkgParams := finance.AmericanOptionParams{
		S0:    params.S0,
		K:     params.K,
		T:     params.T,
		R:     params.R,
		Sigma: params.Sigma,
		IsPut: params.IsPut,
		Paths: params.Paths,
		Steps: params.Steps,
	}
	return p.impl.ComputePrice(pkgParams)
}

// End of Longstaff-Schwartz implementation
