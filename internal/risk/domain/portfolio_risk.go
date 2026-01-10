package domain

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/shopspring/decimal"
	algorithm "github.com/wyfcoding/pkg/algorithm/math"
)

// PortfolioAsset 组合中的单项资产
type PortfolioAsset struct {
	Symbol         string          `json:"symbol"`
	Position       decimal.Decimal `json:"position"`        // 持仓数量 (+为多, -为空)
	CurrentPrice   decimal.Decimal `json:"current_price"`   // 当前单价
	Volatility     float64         `json:"volatility"`      // 年化波动率 (sigma)
	ExpectedReturn float64         `json:"expected_return"` // 预期年化收益率 (mu)
}

// PortfolioRiskInput 组合风险计算输入
type PortfolioRiskInput struct {
	Assets            []PortfolioAsset `json:"assets"`
	CorrelationMatrix [][]float64      `json:"correlation_matrix"` // 相关系数矩阵
	TimeHorizon       float64          `json:"time_horizon"`       // 时间跨度 (年), e.g., 1/252 for 1 day
	Simulations       int              `json:"simulations"`        // 模拟次数
	ConfidenceLevel   float64          `json:"confidence_level"`   // 置信度, e.g., 0.95, 0.99
}

// PortfolioRiskResult 组合风险计算结果
type PortfolioRiskResult struct {
	TotalValue      decimal.Decimal            `json:"total_value"`
	VaR             decimal.Decimal            `json:"var"`             // Value at Risk
	ES              decimal.Decimal            `json:"es"`              // Expected Shortfall (CVaR)
	ComponentVaR    map[string]decimal.Decimal `json:"component_var"`   // 各资产对总风险的贡献 (Marginal VaR * Weight)
	Diversification decimal.Decimal            `json:"diversification"` // 分散化效应带来的风险降低值
}

// CalculatePortfolioRisk 执行多资产关联蒙特卡洛模拟
func CalculatePortfolioRisk(input PortfolioRiskInput) (*PortfolioRiskResult, error) {
	nAssets := len(input.Assets)
	if nAssets == 0 {
		return nil, fmt.Errorf("no assets in portfolio")
	}
	if len(input.CorrelationMatrix) != nAssets || len(input.CorrelationMatrix[0]) != nAssets {
		return nil, fmt.Errorf("correlation matrix dimension mismatch")
	}

	// 1. 构建协方差矩阵 Sigma
	// Cov(i, j) = Rho(i, j) * Sigma(i) * Sigma(j)
	covData := make([][]float64, nAssets)
	for i := range nAssets {
		covData[i] = make([]float64, nAssets)
		for j := range nAssets {
			volI := input.Assets[i].Volatility
			volJ := input.Assets[j].Volatility
			rho := input.CorrelationMatrix[i][j]
			// 调整为时间跨度内的协方差: Cov * T
			// 注意：通常模拟是基于收益率，这里直接对数收益率模拟
			// sigma_scaled = sigma * sqrt(T)
			// cov_scaled = rho * (sigma_i * sqrt(T)) * (sigma_j * sqrt(T)) = rho * sigma_i * sigma_j * T
			covData[i][j] = rho * volI * volJ * input.TimeHorizon
		}
	}

	// 2. Cholesky 分解
	covMatrix, err := algorithm.NewMatrixFromData(covData)
	if err != nil {
		return nil, fmt.Errorf("failed to create covariance matrix: %w", err)
	}
	L, err := covMatrix.Cholesky()
	if err != nil {
		return nil, fmt.Errorf("cholesky decomposition failed (matrix might not be positive definite): %w", err)
	}

	// 3. 蒙特卡洛模拟
	randSource := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))

	// 初始组合价值
	var initialTotalValue decimal.Decimal
	assetValues := make([]float64, nAssets)
	for i, asset := range input.Assets {
		val := asset.Position.Mul(asset.CurrentPrice)
		initialTotalValue = initialTotalValue.Add(val)
		assetValues[i] = val.InexactFloat64()
	}
	initialValFloat := initialTotalValue.InexactFloat64()

	portfolioPnLs := make([]float64, input.Simulations)

	// 漂移项 (Drift) = (mu - 0.5 * sigma^2) * T
	drifts := make([]float64, nAssets)
	for i := range nAssets {
		mu := input.Assets[i].ExpectedReturn
		sig := input.Assets[i].Volatility
		drifts[i] = (mu - 0.5*sig*sig) * input.TimeHorizon
	}

	for s := 0; s < input.Simulations; s++ {
		// 生成 n 个独立标准正态分布随机数
		z := make([]float64, nAssets)
		for i := range nAssets {
			z[i] = randSource.NormFloat64()
		}

		// 关联随机数 x = L * z
		x, _ := L.MultiplyVector(z) // x 现在是服从 Cov 分布的随机变量 (correlated returns)

		// 计算模拟后的资产价格/价值
		var simPortfolioVal float64
		for i := range nAssets {
			// S_T = S_0 * exp(drift + x)
			// 注意：这里 x 已经是 correlated volatility part: sigma * sqrt(T) * epsilon
			// 因为 Cov 矩阵在构建时已经乘以了 T，所以 L 分解出来的 scale 已经包含了 sqrt(T)

			returnRate := math.Exp(drifts[i] + x[i])
			simAssetVal := assetValues[i] * returnRate
			simPortfolioVal += simAssetVal
		}

		portfolioPnLs[s] = simPortfolioVal - initialValFloat
	}

	// 4. 计算统计量
	slices.Sort(portfolioPnLs)

	// VaR
	idx := max(int(math.Floor(float64(input.Simulations)*(1-input.ConfidenceLevel))), 0)
	if idx >= input.Simulations {
		idx = input.Simulations - 1
	}
	varValue := -portfolioPnLs[idx] // VaR 通常表示为正数损失

	// ES (Expected Shortfall) - 尾部平均损失
	sumTail := 0.0
	countTail := 0
	for i := 0; i <= idx; i++ {
		sumTail += portfolioPnLs[i]
		countTail++
	}
	esValue := -sumTail / float64(countTail)

	// 简单计算 Component VaR (近似)
	// 在复杂系统中需要记录每次模拟各资产的 PnL 并进行回归分析
	// 这里暂留空或仅做总和计算

	return &PortfolioRiskResult{
		TotalValue:      initialTotalValue,
		VaR:             decimal.NewFromFloat(math.Max(0, varValue)),
		ES:              decimal.NewFromFloat(math.Max(0, esValue)),
		ComponentVaR:    make(map[string]decimal.Decimal), // 需进一步实现
		Diversification: decimal.Zero,                     // 需计算 Undiversified VaR 后相减
	}, nil
}
