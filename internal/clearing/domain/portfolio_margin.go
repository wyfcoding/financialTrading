//go:build clearing_experimental
// +build clearing_experimental

package domain

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// PortfolioMarginModel 投资组合保证金模型
type PortfolioMarginModel string

const (
	ModelSPAN PortfolioMarginModel = "SPAN" // 标准投资组合风险分析
	ModelTIMS PortfolioMarginModel = "TIMS" // 理论跨保证金系统
	ModelCORE PortfolioMarginModel = "CORE" // 核心投资组合模型
	ModelVAR  PortfolioMarginModel = "VAR"  // 风险价值模型
)

// SPANCalculator SPAN保证金计算器
type SPANCalculator struct {
	riskArrayRepo RiskArrayRepository
	scenarioRepo  ScenarioRepository
	mu            sync.RWMutex
	config        *SPANConfig
}

// SPANConfig SPAN配置
type SPANConfig struct {
	ScanningRange        float64 `json:"scanning_range"`         // 扫描范围
	IntraCommoditySpread float64 `json:"intra_commodity_spread"` // 商品内价差
	InterCommoditySpread float64 `json:"inter_commodity_spread"` // 商品间价差
	ShortOptionMinimum   float64 `json:"short_option_minimum"`   // 空头期权最低保证金
	VolatilityScanRange  float64 `json:"volatility_scan_range"`  // 波动率扫描范围
}

// NewSPANCalculator 创建SPAN计算器
func NewSPANCalculator(riskArrayRepo RiskArrayRepository, scenarioRepo ScenarioRepository) *SPANCalculator {
	return &SPANCalculator{
		riskArrayRepo: riskArrayRepo,
		scenarioRepo:  scenarioRepo,
		config: &SPANConfig{
			ScanningRange:        0.03, // 3%
			IntraCommoditySpread: 0.3,
			InterCommoditySpread: 0.5,
			ShortOptionMinimum:   0.01,
			VolatilityScanRange:  0.01,
		},
	}
}

// CalculateMargin 计算保证金
func (sc *SPANCalculator) CalculateMargin(ctx context.Context, portfolio *Portfolio) (*MarginResult, error) {
	// 获取风险阵列
	riskArrays, err := sc.getRiskArrays(ctx, portfolio)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk arrays: %w", err)
	}

	// 获取情景
	scenarios, err := sc.getScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenarios: %w", err)
	}

	// 计算扫描风险
	scanningRisk := sc.calculateScanningRisk(portfolio, riskArrays, scenarios)

	// 计算价差信用
	spreadCredit := sc.calculateSpreadCredit(portfolio, riskArrays)

	// 计算空头期权最低保证金
	shortOptionMinimum := sc.calculateShortOptionMinimum(portfolio)

	// 计算净保证金
	netMargin := scanningRisk - spreadCredit + shortOptionMinimum

	// 确保保证金非负
	if netMargin < 0 {
		netMargin = 0
	}

	result := &MarginResult{
		PortfolioID:      portfolio.ID,
		TotalMarketValue: sc.calculateTotalMarketValue(portfolio),
		NetMargin:        netMargin,
		Components: []*MarginComponent{
			{
				PositionID:        "SCANNING_RISK",
				Symbol:            "SCANNING_RISK",
				MarginRequirement: scanningRisk,
			},
			{
				PositionID:        "SPREAD_CREDIT",
				Symbol:            "SPREAD_CREDIT",
				MarginRequirement: -spreadCredit,
			},
			{
				PositionID:        "SHORT_OPTION_MIN",
				Symbol:            "SHORT_OPTION_MIN",
				MarginRequirement: shortOptionMinimum,
			},
		},
		CalculatedAt: time.Now(),
	}

	return result, nil
}

// getRiskArrays 获取风险阵列
func (sc *SPANCalculator) getRiskArrays(ctx context.Context, portfolio *Portfolio) ([]*RiskArray, error) {
	var riskArrays []*RiskArray

	for _, position := range portfolio.Positions {
		riskArray, err := sc.riskArrayRepo.GetRiskArray(ctx, position.Symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get risk array for %s: %w", position.Symbol, err)
		}
		riskArrays = append(riskArrays, riskArray)
	}

	return riskArrays, nil
}

// getScenarios 获取情景
func (sc *SPANCalculator) getScenarios(ctx context.Context) ([]*Scenario, error) {
	// 获取标准SPAN情景
	scenarios, err := sc.scenarioRepo.GetSPANScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SPAN scenarios: %w", err)
	}

	return scenarios, nil
}

// calculateScanningRisk 计算扫描风险
func (sc *SPANCalculator) calculateScanningRisk(portfolio *Portfolio, riskArrays []*RiskArray, scenarios []*Scenario) float64 {
	var maxLoss float64

	// 对每个情景计算损失
	for _, scenario := range scenarios {
		scenarioLoss := sc.calculateScenarioLoss(portfolio, riskArrays, scenario)

		if scenarioLoss > maxLoss {
			maxLoss = scenarioLoss
		}
	}

	return maxLoss
}

// calculateScenarioLoss 计算情景损失
func (sc *SPANCalculator) calculateScenarioLoss(portfolio *Portfolio, riskArrays []*RiskArray, scenario *Scenario) float64 {
	var totalLoss float64

	// 计算每个持仓的损失
	for i, position := range portfolio.Positions {
		riskArray := riskArrays[i]

		// 根据情景获取风险因子
		riskFactor := sc.getRiskFactor(riskArray, scenario)

		// 计算持仓损失
		positionLoss := position.MarketValue * riskFactor
		totalLoss += positionLoss
	}

	return totalLoss
}

// getRiskFactor 获取风险因子
func (sc *SPANCalculator) getRiskFactor(riskArray *RiskArray, scenario *Scenario) float64 {
	// 根据情景类型获取相应的风险因子
	switch scenario.Type {
	case "PRICE_UP":
		return riskArray.PriceUp
	case "PRICE_DOWN":
		return riskArray.PriceDown
	case "VOL_UP":
		return riskArray.VolUp
	case "VOL_DOWN":
		return riskArray.VolDown
	case "TIME_DECAY":
		return riskArray.TimeDecay
	default:
		return 0
	}
}

// calculateSpreadCredit 计算价差信用
func (sc *SPANCalculator) calculateSpreadCredit(portfolio *Portfolio, riskArrays []*RiskArray) float64 {
	var totalCredit float64

	// 识别价差关系
	spreads := sc.identifySpreads(portfolio, riskArrays)

	// 计算每个价差的信用
	for _, spread := range spreads {
		credit := sc.calculateSpreadCreditForPair(spread)
		totalCredit += credit
	}

	return totalCredit
}

// identifySpreads 识别价差关系
func (sc *SPANCalculator) identifySpreads(portfolio *Portfolio, riskArrays []*RiskArray) []*SpreadPair {
	var spreads []*SpreadPair

	// 简化实现：识别相同商品的不同月份
	positionsByCommodity := make(map[string][]*Position)

	for _, position := range portfolio.Positions {
		commodity := sc.extractCommodity(position.Symbol)
		positionsByCommodity[commodity] = append(positionsByCommodity[commodity], position)
	}

	// 为每个商品识别价差
	for _, positions := range positionsByCommodity {
		if len(positions) >= 2 {
			// 创建价差对
			for i := 0; i < len(positions)-1; i++ {
				for j := i + 1; j < len(positions); j++ {
					spread := &SpreadPair{
						Position1:  positions[i],
						Position2:  positions[j],
						SpreadType: "INTRA_COMMODITY",
					}
					spreads = append(spreads, spread)
				}
			}
		}
	}

	return spreads
}

// extractCommodity 提取商品
func (sc *SPANCalculator) extractCommodity(symbol string) string {
	// 简化实现：假设符号的前几个字符是商品代码
	if len(symbol) >= 2 {
		return symbol[:2]
	}
	return symbol
}

// calculateSpreadCreditForPair 计算价对信用
func (sc *SPANCalculator) calculateSpreadCreditForPair(spread *SpreadPair) float64 {
	// 计算价差大小
	spreadSize := math.Abs(spread.Position1.Price - spread.Position2.Price)

	// 根据价差类型计算信用
	var creditRate float64
	if spread.SpreadType == "INTRA_COMMODITY" {
		creditRate = sc.config.IntraCommoditySpread
	} else {
		creditRate = sc.config.InterCommoditySpread
	}

	// 信用 = 价差大小 * 信用率
	credit := spreadSize * creditRate

	return credit
}

// calculateShortOptionMinimum 计算空头期权最低保证金
func (sc *SPANCalculator) calculateShortOptionMinimum(portfolio *Portfolio) float64 {
	var totalMinimum float64

	for _, position := range portfolio.Positions {
		// 检查是否为空头期权
		if sc.isShortOption(position) {
			minimum := position.MarketValue * sc.config.ShortOptionMinimum
			totalMinimum += minimum
		}
	}

	return totalMinimum
}

// isShortOption 是否为空头期权
func (sc *SPANCalculator) isShortOption(position *Position) bool {
	// 简化实现：检查符号是否包含期权特征
	// 实际应该根据期权类型和方向判断
	return false
}

// calculateTotalMarketValue 计算总市值
func (sc *SPANCalculator) calculateTotalMarketValue(portfolio *Portfolio) float64 {
	var total float64
	for _, position := range portfolio.Positions {
		total += position.MarketValue
	}
	return total
}

// TIMSCalculator TIMS保证金计算器
type TIMSCalculator struct {
	correlationRepo CorrelationRepository
	volatilityRepo  VolatilityRepository
	mu              sync.RWMutex
	config          *TIMSConfig
}

// TIMSConfig TIMS配置
type TIMSConfig struct {
	ConfidenceLevel       float64 `json:"confidence_level"`
	TimeHorizon           int     `json:"time_horizon"` // 天数
	DiversificationFactor float64 `json:"diversification_factor"`
	LiquidityFactor       float64 `json:"liquidity_factor"`
	ConcentrationFactor   float64 `json:"concentration_factor"`
}

// NewTIMSCalculator 创建TIMS计算器
func NewTIMSCalculator(correlationRepo CorrelationRepository, volatilityRepo VolatilityRepository) *TIMSCalculator {
	return &TIMSCalculator{
		correlationRepo: correlationRepo,
		volatilityRepo:  volatilityRepo,
		config: &TIMSConfig{
			ConfidenceLevel:       0.99,
			TimeHorizon:           1,
			DiversificationFactor: 0.7,
			LiquidityFactor:       1.0,
			ConcentrationFactor:   1.2,
		},
	}
}

// CalculateMargin 计算保证金
func (tc *TIMSCalculator) CalculateMargin(ctx context.Context, portfolio *Portfolio) (*MarginResult, error) {
	// 获取波动率数据
	volatilities, err := tc.getVolatilities(ctx, portfolio)
	if err != nil {
		return nil, fmt.Errorf("failed to get volatilities: %w", err)
	}

	// 获取相关性矩阵
	correlationMatrix, err := tc.getCorrelationMatrix(ctx, portfolio)
	if err != nil {
		return nil, fmt.Errorf("failed to get correlation matrix: %w", err)
	}

	// 计算投资组合方差
	portfolioVariance := tc.calculatePortfolioVariance(portfolio, volatilities, correlationMatrix)

	// 计算投资组合标准差
	portfolioStdDev := math.Sqrt(math.Abs(portfolioVariance))

	// 计算风险价值
	varValue := portfolioStdDev * tc.getZScore(tc.config.ConfidenceLevel)

	// 应用调整因子
	adjustedMargin := varValue * tc.config.DiversificationFactor *
		tc.config.LiquidityFactor * tc.config.ConcentrationFactor

	result := &MarginResult{
		PortfolioID:      portfolio.ID,
		TotalMarketValue: tc.calculateTotalMarketValue(portfolio),
		NetMargin:        adjustedMargin,
		Components: []*MarginComponent{
			{
				PositionID:        "VAR",
				Symbol:            "VAR",
				MarginRequirement: varValue,
			},
			{
				PositionID:        "DIVERSIFICATION",
				Symbol:            "DIVERSIFICATION",
				MarginRequirement: varValue * (tc.config.DiversificationFactor - 1),
			},
			{
				PositionID:        "LIQUIDITY",
				Symbol:            "LIQUIDITY",
				MarginRequirement: varValue * (tc.config.LiquidityFactor - 1),
			},
			{
				PositionID:        "CONCENTRATION",
				Symbol:            "CONCENTRATION",
				MarginRequirement: varValue * (tc.config.ConcentrationFactor - 1),
			},
		},
		CalculatedAt: time.Now(),
	}

	return result, nil
}

// getVolatilities 获取波动率
func (tc *TIMSCalculator) getVolatilities(ctx context.Context, portfolio *Portfolio) (map[string]float64, error) {
	volatilities := make(map[string]float64)

	for _, position := range portfolio.Positions {
		volatility, err := tc.volatilityRepo.GetVolatility(ctx, position.Symbol, tc.config.TimeHorizon)
		if err != nil {
			return nil, fmt.Errorf("failed to get volatility for %s: %w", position.Symbol, err)
		}
		volatilities[position.Symbol] = volatility
	}

	return volatilities, nil
}

// getCorrelationMatrix 获取相关性矩阵
func (tc *TIMSCalculator) getCorrelationMatrix(ctx context.Context, portfolio *Portfolio) (map[string]map[string]float64, error) {
	matrix := make(map[string]map[string]float64)

	// 获取所有符号
	var symbols []string
	for _, position := range portfolio.Positions {
		symbols = append(symbols, position.Symbol)
	}

	// 获取相关性数据
	correlations, err := tc.correlationRepo.GetCorrelations(ctx, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to get correlations: %w", err)
	}

	// 构建矩阵
	for i, symbol1 := range symbols {
		matrix[symbol1] = make(map[string]float64)

		for j, symbol2 := range symbols {
			if i == j {
				matrix[symbol1][symbol2] = 1.0
			} else {
				// 查找相关性
				corr := tc.findCorrelation(correlations, symbol1, symbol2)
				matrix[symbol1][symbol2] = corr
			}
		}
	}

	return matrix, nil
}

// findCorrelation 查找相关性
func (tc *TIMSCalculator) findCorrelation(correlations []*Correlation, symbol1, symbol2 string) float64 {
	for _, corr := range correlations {
		if (corr.Symbol1 == symbol1 && corr.Symbol2 == symbol2) ||
			(corr.Symbol1 == symbol2 && corr.Symbol2 == symbol1) {
			return corr.Correlation
		}
	}

	// 默认相关性
	return 0.3
}

// calculatePortfolioVariance 计算投资组合方差
func (tc *TIMSCalculator) calculatePortfolioVariance(portfolio *Portfolio,
	volatilities map[string]float64, correlationMatrix map[string]map[string]float64) float64 {

	var variance float64

	for _, pos1 := range portfolio.Positions {
		for _, pos2 := range portfolio.Positions {
			weight1 := pos1.MarketValue * volatilities[pos1.Symbol]
			weight2 := pos2.MarketValue * volatilities[pos2.Symbol]
			corr := correlationMatrix[pos1.Symbol][pos2.Symbol]

			variance += weight1 * weight2 * corr
		}
	}

	return variance
}

// getZScore 获取Z分数
func (tc *TIMSCalculator) getZScore(confidenceLevel float64) float64 {
	// 标准正态分布Z分数
	zScores := map[float64]float64{
		0.90:  1.282,
		0.95:  1.645,
		0.99:  2.326,
		0.995: 2.576,
		0.999: 3.090,
	}

	if z, exists := zScores[confidenceLevel]; exists {
		return z
	}

	// 默认值
	return 2.326 // 99%置信度
}

// calculateTotalMarketValue 计算总市值
func (tc *TIMSCalculator) calculateTotalMarketValue(portfolio *Portfolio) float64 {
	var total float64
	for _, position := range portfolio.Positions {
		total += position.MarketValue
	}
	return total
}

// PortfolioMarginOptimizer 投资组合保证金优化器
type PortfolioMarginOptimizer struct {
	marginCalculators map[PortfolioMarginModel]MarginCalculator
	mu                sync.RWMutex
}

// NewPortfolioMarginOptimizer 创建投资组合保证金优化器
func NewPortfolioMarginOptimizer() *PortfolioMarginOptimizer {
	return &PortfolioMarginOptimizer{
		marginCalculators: make(map[PortfolioMarginModel]MarginCalculator),
	}
}

// RegisterCalculator 注册计算器
func (pmo *PortfolioMarginOptimizer) RegisterCalculator(model PortfolioMarginModel, calculator MarginCalculator) {
	pmo.mu.Lock()
	defer pmo.mu.Unlock()

	pmo.marginCalculators[model] = calculator
}

// OptimizeMargin 优化保证金
func (pmo *PortfolioMarginOptimizer) OptimizeMargin(ctx context.Context, portfolio *Portfolio) (*OptimizationResult, error) {
	pmo.mu.RLock()
	defer pmo.mu.RUnlock()

	if len(pmo.marginCalculators) == 0 {
		return nil, fmt.Errorf("no margin calculators registered")
	}

	// 使用所有模型计算保证金
	var results []*ModelResult

	for model, calculator := range pmo.marginCalculators {
		marginResult, err := calculator.CalculateMargin(ctx, portfolio)
		if err != nil {
			fmt.Printf("Failed to calculate margin with model %s: %v\n", model, err)
			continue
		}

		modelResult := &ModelResult{
			Model:           model,
			MarginResult:    marginResult,
			CalculationTime: time.Now(),
		}
		results = append(results, modelResult)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all margin calculations failed")
	}

	// 选择最低保证金
	bestResult := results[0]
	for _, result := range results[1:] {
		if result.MarginResult.NetMargin < bestResult.MarginResult.NetMargin {
			bestResult = result
		}
	}

	optimizationResult := &OptimizationResult{
		PortfolioID:     portfolio.ID,
		BestModel:       bestResult.Model,
		OptimizedMargin: bestResult.MarginResult.NetMargin,
		AllResults:      results,
		OptimizedAt:     time.Now(),
	}

	return optimizationResult, nil
}

// Data structures

type RiskArray struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	PriceUp   float64   `json:"price_up"`
	PriceDown float64   `json:"price_down"`
	VolUp     float64   `json:"vol_up"`
	VolDown   float64   `json:"vol_down"`
	TimeDecay float64   `json:"time_decay"`
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
}

type Scenario struct {
	ID           string  `json:"id"`
	ScenarioName string  `json:"scenario_name"`
	Type         string  `json:"type"` // PRICE_UP, PRICE_DOWN, VOL_UP, VOL_DOWN, TIME_DECAY
	Description  string  `json:"description"`
	Probability  float64 `json:"probability"`
	Weight       float64 `json:"weight"`
}

type SpreadPair struct {
	Position1  *Position `json:"position1"`
	Position2  *Position `json:"position2"`
	SpreadType string    `json:"spread_type"` // INTRA_COMMODITY, INTER_COMMODITY
}

type ModelResult struct {
	Model           PortfolioMarginModel `json:"model"`
	MarginResult    *MarginResult        `json:"margin_result"`
	CalculationTime time.Time            `json:"calculation_time"`
}

type OptimizationResult struct {
	PortfolioID     string               `json:"portfolio_id"`
	BestModel       PortfolioMarginModel `json:"best_model"`
	OptimizedMargin float64              `json:"optimized_margin"`
	AllResults      []*ModelResult       `json:"all_results"`
	OptimizedAt     time.Time            `json:"optimized_at"`
}

// Repository interfaces

type RiskArrayRepository interface {
	GetRiskArray(ctx context.Context, symbol string) (*RiskArray, error)
	GetRiskArrays(ctx context.Context, symbols []string) ([]*RiskArray, error)
	SaveRiskArray(ctx context.Context, riskArray *RiskArray) error
	UpdateRiskArray(ctx context.Context, riskArray *RiskArray) error
}

type ScenarioRepository interface {
	GetSPANScenarios(ctx context.Context) ([]*Scenario, error)
	GetScenario(ctx context.Context, scenarioID string) (*Scenario, error)
	SaveScenario(ctx context.Context, scenario *Scenario) error
	UpdateScenario(ctx context.Context, scenario *Scenario) error
}

type VolatilityRepository interface {
	GetVolatility(ctx context.Context, symbol string, horizon int) (float64, error)
	GetHistoricalVolatility(ctx context.Context, symbol string, startDate, endDate time.Time) ([]float64, error)
	SaveVolatility(ctx context.Context, symbol string, volatility float64, horizon int) error
}

type MarginCalculator interface {
	CalculateMargin(ctx context.Context, portfolio *Portfolio) (*MarginResult, error)
}
