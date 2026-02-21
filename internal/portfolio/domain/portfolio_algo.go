// 变更说明：
// 从 pkg/algos/finance/portfolio.go 迁移。
// 实现了均值-方差组合优化与风险计算。
package domain

import (
	"math"

	"github.com/shopspring/decimal"
	algomath "github.com/wyfcoding/pkg/algos/math"
)

// PortfolioOptimizer 组合优化器
type PortfolioOptimizer struct {
	Assets     []string
	Returns    []decimal.Decimal   // 预期收益率
	Covariance [][]decimal.Decimal // 协方差矩阵
}

func NewPortfolioOptimizer(assets []string, returns []decimal.Decimal, cov [][]decimal.Decimal) *PortfolioOptimizer {
	return &PortfolioOptimizer{
		Assets:     assets,
		Returns:    returns,
		Covariance: cov,
	}
}

// OptimizeMinimumVariance 最小方差组合优化
func (o *PortfolioOptimizer) OptimizeMinimumVariance() map[string]decimal.Decimal {
	n := len(o.Assets)
	if n == 0 {
		return nil
	}

	matrixData := make([][]float64, n)
	for i := range o.Covariance {
		matrixData[i] = make([]float64, n)
		for j := range o.Covariance[i] {
			matrixData[i][j] = o.Covariance[i][j].InexactFloat64()
		}
	}

	sigma, err := algomath.NewMatrixFromData(matrixData)
	if err != nil {
		return o.EqualWeight()
	}

	ones := make([]float64, n)
	for i := range ones {
		ones[i] = 1.0
	}

	wRaw, err := sigma.SolveCholesky(ones)
	if err != nil {
		return o.EqualWeight()
	}

	sumWRaw := 0.0
	for _, w := range wRaw {
		sumWRaw += w
	}

	weights := make(map[string]decimal.Decimal)
	for i, asset := range o.Assets {
		weights[asset] = decimal.NewFromFloat(wRaw[i] / sumWRaw)
	}

	return weights
}

// EqualWeight 等权重分配
func (o *PortfolioOptimizer) EqualWeight() map[string]decimal.Decimal {
	n := len(o.Assets)
	weight := decimal.NewFromFloat(1.0 / float64(n))
	weights := make(map[string]decimal.Decimal)
	for _, asset := range o.Assets {
		weights[asset] = weight
	}
	return weights
}

// CalculatePortfolioRisk 计算组合风险 (标准差)
func (o *PortfolioOptimizer) CalculatePortfolioRisk(weights map[string]decimal.Decimal) decimal.Decimal {
	var variance float64
	for i, a1 := range o.Assets {
		w1 := weights[a1].InexactFloat64()
		for j, a2 := range o.Assets {
			w2 := weights[a2].InexactFloat64()
			cov := o.Covariance[i][j].InexactFloat64()
			variance += w1 * w2 * cov
		}
	}
	return decimal.NewFromFloat(math.Sqrt(variance))
}

// CalculateReturns 计算收益率序列
func CalculateReturns(prices []decimal.Decimal) []decimal.Decimal {
	if len(prices) < 2 {
		return nil
	}
	returns := make([]decimal.Decimal, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if !prices[i-1].IsZero() {
			returns[i-1] = prices[i].Sub(prices[i-1]).Div(prices[i-1])
		}
	}
	return returns
}

// CalculateCovariance 计算协方差矩阵
func CalculateCovariance(assetsReturns [][]decimal.Decimal) [][]decimal.Decimal {
	n := len(assetsReturns)
	if n == 0 {
		return nil
	}
	m := len(assetsReturns[0])
	cov := make([][]decimal.Decimal, n)
	for i := 0; i < n; i++ {
		cov[i] = make([]decimal.Decimal, n)
	}

	means := make([]float64, n)
	for i := 0; i < n; i++ {
		var sum float64
		for _, r := range assetsReturns[i] {
			sum += r.InexactFloat64()
		}
		means[i] = sum / float64(m)
	}

	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			var sum float64
			for k := 0; k < m; k++ {
				diffI := assetsReturns[i][k].InexactFloat64() - means[i]
				diffJ := assetsReturns[j][k].InexactFloat64() - means[j]
				sum += diffI * diffJ
			}
			val := decimal.NewFromFloat(sum / float64(m-1))
			cov[i][j] = val
			cov[j][i] = val
		}
	}

	return cov
}
