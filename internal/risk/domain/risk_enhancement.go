//go:build risk_experimental
// +build risk_experimental

package domain

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// RiskCategory 风险类别
type RiskCategory string

const (
	RiskMarket      RiskCategory = "MARKET"      // 市场风险
	RiskCredit      RiskCategory = "CREDIT"      // 信用风险
	RiskLiquidity   RiskCategory = "LIQUIDITY"   // 流动性风险
	RiskOperational RiskCategory = "OPERATIONAL" // 操作风险
	RiskLegal       RiskCategory = "LEGAL"       // 法律风险
	RiskSystemic    RiskCategory = "SYSTEMIC"    // 系统性风险
)

// RiskMetric 风险指标
type RiskMetric string

const (
	MetricVaR        RiskMetric = "VAR"         // 风险价值
	MetricCVaR       RiskMetric = "CVAR"        // 条件风险价值
	MetricStressTest RiskMetric = "STRESS_TEST" // 压力测试
	MetricScenario   RiskMetric = "SCENARIO"    // 情景分析
	MetricBacktest   RiskMetric = "BACKTEST"    // 回测
)

// RiskModel 风险模型
type RiskModel string

const (
	ModelHistoricalSimulation RiskModel = "HISTORICAL_SIMULATION" // 历史模拟法
	ModelParametric           RiskModel = "PARAMETRIC"            // 参数法
	ModelMonteCarlo           RiskModel = "MONTE_CARLO"           // 蒙特卡洛模拟
	ModelGARCH                RiskModel = "GARCH"                 // GARCH模型
)

// RiskAssessment 风险评估
type RiskAssessment struct {
	ID                string                        `json:"id"`
	Timestamp         time.Time                     `json:"timestamp"`
	PortfolioID       string                        `json:"portfolio_id"`
	RiskCategory      RiskCategory                  `json:"risk_category"`
	RiskMetric        RiskMetric                    `json:"risk_metric"`
	RiskModel         RiskModel                     `json:"risk_model"`
	ConfidenceLevel   float64                       `json:"confidence_level"`   // 置信水平 (0-1)
	TimeHorizon       time.Duration                 `json:"time_horizon"`       // 时间期限
	VaR               float64                       `json:"var"`                // 风险价值
	CVaR              float64                       `json:"cvar"`               // 条件风险价值
	StressLoss        float64                       `json:"stress_loss"`        // 压力测试损失
	ExpectedLoss      float64                       `json:"expected_loss"`      // 预期损失
	UnexpectedLoss    float64                       `json:"unexpected_loss"`    // 非预期损失
	RiskContributions map[string]float64            `json:"risk_contributions"` // 风险贡献度
	CorrelationMatrix map[string]map[string]float64 `json:"correlation_matrix"` // 相关性矩阵
	Metadata          map[string]interface{}        `json:"metadata"`
}

// RiskLimit 风险限额
type RiskLimit struct {
	ID               string                 `json:"id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LimitName        string                 `json:"limit_name"`
	RiskCategory     RiskCategory           `json:"risk_category"`
	Metric           RiskMetric             `json:"metric"`
	Symbol           string                 `json:"symbol"`
	LimitValue       float64                `json:"limit_value"`
	WarningThreshold float64                `json:"warning_threshold"` // 预警阈值
	HardLimit        bool                   `json:"hard_limit"`        // 硬限额
	Enabled          bool                   `json:"enabled"`
	TimeWindow       time.Duration          `json:"time_window"`     // 时间窗口
	ResetFrequency   time.Duration          `json:"reset_frequency"` // 重置频率
	Conditions       map[string]interface{} `json:"conditions"`
	Description      string                 `json:"description"`
}

// RiskEvent 风险事件
type RiskEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	Severity     string                 `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	RiskCategory RiskCategory           `json:"risk_category"`
	Symbol       string                 `json:"symbol"`
	OrderID      string                 `json:"order_id"`
	PortfolioID  string                 `json:"portfolio_id"`
	Description  string                 `json:"description"`
	Impact       float64                `json:"impact"`      // 影响程度
	Probability  float64                `json:"probability"` // 发生概率
	Status       string                 `json:"status"`      // OPEN, INVESTIGATING, RESOLVED, CLOSED
	AssignedTo   string                 `json:"assigned_to"`
	Resolution   string                 `json:"resolution"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RiskIndicator 风险指标
type RiskIndicator struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	IndicatorName string                 `json:"indicator_name"`
	RiskCategory  RiskCategory           `json:"risk_category"`
	Symbol        string                 `json:"symbol"`
	Value         float64                `json:"value"`
	Threshold     float64                `json:"threshold"`
	Status        string                 `json:"status"` // NORMAL, WARNING, ALERT
	Trend         string                 `json:"trend"`  // UP, DOWN, STABLE
	Volatility    float64                `json:"volatility"`
	Correlation   float64                `json:"correlation"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RiskManager 风险管理器
type RiskManager struct {
	riskRepo       RiskRepository
	limitRepo      RiskLimitRepository
	eventRepo      RiskEventRepository
	indicatorRepo  RiskIndicatorRepository
	marketDataRepo MarketDataRepository
	mu             sync.RWMutex
	limits         map[string]*RiskLimit
	indicators     map[string]*RiskIndicator
	riskModels     map[RiskModel]RiskModelCalculator
}

// NewRiskManager 创建风险管理器
func NewRiskManager(
	riskRepo RiskRepository,
	limitRepo RiskLimitRepository,
	eventRepo RiskEventRepository,
	indicatorRepo RiskIndicatorRepository,
	marketDataRepo MarketDataRepository,
) *RiskManager {
	return &RiskManager{
		riskRepo:       riskRepo,
		limitRepo:      limitRepo,
		eventRepo:      eventRepo,
		indicatorRepo:  indicatorRepo,
		marketDataRepo: marketDataRepo,
		limits:         make(map[string]*RiskLimit),
		indicators:     make(map[string]*RiskIndicator),
		riskModels:     make(map[RiskModel]RiskModelCalculator),
	}
}

// Initialize 初始化风险管理器
func (rm *RiskManager) Initialize(ctx context.Context) error {
	// 加载风险限额
	limits, err := rm.limitRepo.GetEnabledLimits(ctx)
	if err != nil {
		return fmt.Errorf("failed to load risk limits: %w", err)
	}

	rm.mu.Lock()
	for _, limit := range limits {
		rm.limits[limit.ID] = limit
	}
	rm.mu.Unlock()

	// 初始化风险模型
	rm.initializeRiskModels()

	return nil
}

// AssessPortfolioRisk 评估投资组合风险
func (rm *RiskManager) AssessPortfolioRisk(ctx context.Context, portfolioID string,
	riskCategory RiskCategory, metric RiskMetric, model RiskModel) (*RiskAssessment, error) {

	// 获取投资组合数据
	portfolio, err := rm.riskRepo.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	// 获取市场数据
	marketData, err := rm.marketDataRepo.GetHistoricalData(ctx, portfolio.Symbols, 252) // 1年历史数据
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	// 选择风险模型计算器
	calculator, exists := rm.riskModels[model]
	if !exists {
		return nil, fmt.Errorf("unsupported risk model: %s", model)
	}

	// 计算风险指标
	assessment, err := calculator.Calculate(portfolio, marketData, riskCategory, metric)
	if err != nil {
		return nil, fmt.Errorf("risk calculation failed: %w", err)
	}

	// 保存风险评估
	err = rm.riskRepo.SaveAssessment(ctx, assessment)
	if err != nil {
		return nil, fmt.Errorf("failed to save risk assessment: %w", err)
	}

	// 检查风险限额
	limitBreaches, err := rm.checkRiskLimits(ctx, assessment)
	if err != nil {
		return nil, fmt.Errorf("failed to check risk limits: %w", err)
	}

	// 记录风险事件
	for _, breach := range limitBreaches {
		event := &RiskEvent{
			ID:           generateEventID(),
			Timestamp:    time.Now(),
			EventType:    "LIMIT_BREACH",
			Severity:     breach.Severity,
			RiskCategory: riskCategory,
			Symbol:       breach.Symbol,
			PortfolioID:  portfolioID,
			Description: fmt.Sprintf("Risk limit breached: %s = %.2f (limit: %.2f)",
				breach.Metric, breach.Value, breach.Limit),
			Impact:      breach.Impact,
			Probability: breach.Probability,
			Status:      "OPEN",
			Metadata:    breach.Metadata,
		}

		err = rm.eventRepo.SaveEvent(ctx, event)
		if err != nil {
			// 记录错误但不中断流程
			fmt.Printf("Failed to save risk event: %v\n", err)
		}
	}

	return assessment, nil
}

// CheckOrderRisk 检查订单风险
func (rm *RiskManager) CheckOrderRisk(ctx context.Context, order *Order, portfolio *Portfolio) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		OrderID:   order.OrderID,
		Symbol:    order.Symbol,
		Timestamp: time.Now(),
		Passed:    true,
		Reasons:   make([]string, 0),
	}

	// 检查市场风险
	marketRisk, err := rm.checkMarketRisk(ctx, order, portfolio)
	if err != nil {
		return nil, fmt.Errorf("market risk check failed: %w", err)
	}

	if !marketRisk.Passed {
		result.Passed = false
		result.Reasons = append(result.Reasons, marketRisk.Reasons...)
	}

	// 检查信用风险
	creditRisk, err := rm.checkCreditRisk(ctx, order, portfolio)
	if err != nil {
		return nil, fmt.Errorf("credit risk check failed: %w", err)
	}

	if !creditRisk.Passed {
		result.Passed = false
		result.Reasons = append(result.Reasons, creditRisk.Reasons...)
	}

	// 检查流动性风险
	liquidityRisk, err := rm.checkLiquidityRisk(ctx, order, portfolio)
	if err != nil {
		return nil, fmt.Errorf("liquidity risk check failed: %w", err)
	}

	if !liquidityRisk.Passed {
		result.Passed = false
		result.Reasons = append(result.Reasons, liquidityRisk.Reasons...)
	}

	// 检查操作风险
	operationalRisk, err := rm.checkOperationalRisk(ctx, order, portfolio)
	if err != nil {
		return nil, fmt.Errorf("operational risk check failed: %w", err)
	}

	if !operationalRisk.Passed {
		result.Passed = false
		result.Reasons = append(result.Reasons, operationalRisk.Reasons...)
	}

	// 记录风险检查结果
	err = rm.riskRepo.SaveRiskCheck(ctx, result)
	if err != nil {
		return nil, fmt.Errorf("failed to save risk check result: %w", err)
	}

	return result, nil
}

// checkMarketRisk 检查市场风险
func (rm *RiskManager) checkMarketRisk(ctx context.Context, order *Order, portfolio *Portfolio) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:  true,
		Reasons: make([]string, 0),
	}

	// 获取市场数据
	marketData, err := rm.marketDataRepo.GetCurrentData(ctx, order.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	// 检查价格波动性
	volatility := rm.calculateVolatility(marketData.HistoricalPrices)
	if volatility > rm.getLimitValue("MARKET_VOLATILITY", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High volatility: %.2f%%", volatility*100))
	}

	// 检查价差
	spread := marketData.Ask - marketData.Bid
	if spread > rm.getLimitValue("MARKET_SPREAD", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Wide spread: %.4f", spread))
	}

	// 检查交易量
	volumeRatio := marketData.Volume / marketData.AverageVolume
	if volumeRatio > rm.getLimitValue("MARKET_VOLUME_RATIO", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High volume ratio: %.2f", volumeRatio))
	}

	return result, nil
}

// checkCreditRisk 检查信用风险
func (rm *RiskManager) checkCreditRisk(ctx context.Context, order *Order, portfolio *Portfolio) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:  true,
		Reasons: make([]string, 0),
	}

	// 获取交易对手信用评级
	counterpartyRating, err := rm.riskRepo.GetCounterpartyRating(ctx, order.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counterparty rating: %w", err)
	}

	// 检查信用评级
	if counterpartyRating.Score < rm.getLimitValue("CREDIT_RATING_MIN", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Low credit rating: %s (score: %.1f)",
				counterpartyRating.Rating, counterpartyRating.Score))
	}

	// 检查敞口限额
	exposure := rm.calculateExposure(portfolio, order)
	exposureLimit := rm.getLimitValue("CREDIT_EXPOSURE_MAX", order.Symbol)

	if exposure > exposureLimit {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Credit exposure exceeded: %.2f (limit: %.2f)",
				exposure, exposureLimit))
	}

	// 检查集中度风险
	concentration := rm.calculateConcentration(portfolio, order.Symbol)
	concentrationLimit := rm.getLimitValue("CREDIT_CONCENTRATION_MAX", order.Symbol)

	if concentration > concentrationLimit {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High concentration: %.2f%% (limit: %.2f%%)",
				concentration*100, concentrationLimit*100))
	}

	return result, nil
}

// checkLiquidityRisk 检查流动性风险
func (rm *RiskManager) checkLiquidityRisk(ctx context.Context, order *Order, portfolio *Portfolio) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:  true,
		Reasons: make([]string, 0),
	}

	// 检查订单规模相对于市场深度
	orderSizeRatio := order.Quantity / rm.getMarketDepth(order.Symbol)
	if orderSizeRatio > rm.getLimitValue("LIQUIDITY_ORDER_SIZE_RATIO", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Large order relative to market depth: %.2f%%",
				orderSizeRatio*100))
	}

	// 检查买卖价差
	spread := rm.getBidAskSpread(order.Symbol)
	if spread > rm.getLimitValue("LIQUIDITY_SPREAD_MAX", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("Wide bid-ask spread: %.4f", spread))
	}

	// 检查市场冲击成本
	impactCost := rm.estimateMarketImpact(order)
	if impactCost > rm.getLimitValue("LIQUIDITY_IMPACT_MAX", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High market impact cost: %.2f%%", impactCost*100))
	}

	return result, nil
}

// checkOperationalRisk 检查操作风险
func (rm *RiskManager) checkOperationalRisk(ctx context.Context, order *Order, portfolio *Portfolio) (*RiskCheckResult, error) {
	result := &RiskCheckResult{
		Passed:  true,
		Reasons: make([]string, 0),
	}

	// 检查交易频率
	tradeFrequency := rm.getTradeFrequency(order.Symbol)
	if tradeFrequency > rm.getLimitValue("OPERATIONAL_FREQUENCY_MAX", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High trade frequency: %.2f trades/min", tradeFrequency))
	}

	// 检查错误率
	errorRate := rm.getErrorRate(order.Symbol)
	if errorRate > rm.getLimitValue("OPERATIONAL_ERROR_MAX", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High error rate: %.2f%%", errorRate*100))
	}

	// 检查系统延迟
	systemLatency := rm.getSystemLatency()
	if systemLatency > rm.getLimitValue("OPERATIONAL_LATENCY_MAX", order.Symbol) {
		result.Passed = false
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("High system latency: %.2fms", systemLatency))
	}

	return result, nil
}

// checkRiskLimits 检查风险限额
func (rm *RiskManager) checkRiskLimits(ctx context.Context, assessment *RiskAssessment) ([]*LimitBreach, error) {
	var breaches []*LimitBreach

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 检查所有适用的风险限额
	for _, limit := range rm.limits {
		if limit.RiskCategory != assessment.RiskCategory {
			continue
		}

		if limit.Metric != assessment.RiskMetric {
			continue
		}

		// 检查限额是否被突破
		value := rm.getAssessmentValue(assessment, limit.Metric)
		if value > limit.LimitValue {
			breach := &LimitBreach{
				LimitID:     limit.ID,
				LimitName:   limit.LimitName,
				Metric:      string(limit.Metric),
				Value:       value,
				Limit:       limit.LimitValue,
				Symbol:      limit.Symbol,
				Severity:    rm.calculateBreachSeverity(value, limit),
				Impact:      rm.calculateBreachImpact(value, limit),
				Probability: rm.calculateBreachProbability(limit),
				Metadata:    make(map[string]interface{}),
			}

			breaches = append(breaches, breach)
		}
	}

	return breaches, nil
}

// Helper methods

func (rm *RiskManager) initializeRiskModels() {
	// 注册风险模型计算器
	rm.riskModels[ModelHistoricalSimulation] = &HistoricalSimulationCalculator{}
	rm.riskModels[ModelParametric] = &ParametricCalculator{}
	rm.riskModels[ModelMonteCarlo] = &MonteCarloCalculator{}
	rm.riskModels[ModelGARCH] = &GARCHCalculator{}
}

func (rm *RiskManager) getLimitValue(limitName, symbol string) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 查找适用的限额
	for _, limit := range rm.limits {
		if limit.LimitName == limitName && (limit.Symbol == symbol || limit.Symbol == "") {
			return limit.LimitValue
		}
	}

	// 返回默认值
	return getDefaultLimitValue(limitName)
}

func (rm *RiskManager) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算对数收益率
	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := math.Log(prices[i] / prices[i-1])
		returns = append(returns, ret)
	}

	// 计算标准差
	mean := calculateMean(returns)
	variance := 0.0
	for _, ret := range returns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance)
}

func (rm *RiskManager) calculateExposure(portfolio *Portfolio, order *Order) float64 {
	// 计算总敞口
	totalExposure := 0.0

	for _, position := range portfolio.Positions {
		if position.Symbol == order.Symbol {
			// 考虑新订单的影响
			if order.Side == SideBuy {
				totalExposure += position.MarketValue + (order.Quantity * order.Price)
			} else {
				totalExposure += position.MarketValue - (order.Quantity * order.Price)
			}
		} else {
			totalExposure += position.MarketValue
		}
	}

	return totalExposure
}

func (rm *RiskManager) calculateConcentration(portfolio *Portfolio, symbol string) float64 {
	// 计算单一资产的集中度
	totalValue := 0.0
	symbolValue := 0.0

	for _, position := range portfolio.Positions {
		totalValue += position.MarketValue
		if position.Symbol == symbol {
			symbolValue += position.MarketValue
		}
	}

	if totalValue == 0 {
		return 0
	}

	return symbolValue / totalValue
}

func (rm *RiskManager) getMarketDepth(symbol string) float64 {
	marketData, err := rm.marketDataRepo.GetCurrentData(context.Background(), symbol)
	if err != nil || marketData == nil {
		return 1_000_000
	}
	depth := math.Max(marketData.Volume, marketData.AverageVolume)
	if depth <= 0 {
		return 1_000_000
	}
	return depth
}

func (rm *RiskManager) getBidAskSpread(symbol string) float64 {
	marketData, err := rm.marketDataRepo.GetCurrentData(context.Background(), symbol)
	if err != nil || marketData == nil {
		return 0.01
	}
	spread := marketData.Ask - marketData.Bid
	if spread < 0 {
		return 0.01
	}
	return spread
}

func (rm *RiskManager) estimateMarketImpact(order *Order) float64 {
	// 简单估算市场冲击成本
	// 基于订单规模相对于市场深度的比例
	marketDepth := rm.getMarketDepth(order.Symbol)
	sizeRatio := order.Quantity / marketDepth

	// 使用平方根法则估算冲击成本
	return 0.1 * math.Sqrt(sizeRatio)
}

func (rm *RiskManager) getTradeFrequency(symbol string) float64 {
	data, err := rm.marketDataRepo.GetIntradayData(context.Background(), symbol, time.Minute, time.Hour)
	if err != nil || len(data) == 0 {
		return 10
	}
	// 每分钟样本数近似为交易频率。
	return float64(len(data)) / 60.0
}

func (rm *RiskManager) getErrorRate(symbol string) float64 {
	events, err := rm.eventRepo.GetEventsByType(context.Background(), "SYSTEM_ERROR", time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return 0.001
	}
	tradeFrequency := rm.getTradeFrequency(symbol)
	baseTrades := math.Max(tradeFrequency*60, 1)
	errorRate := float64(len(events)) / baseTrades
	if errorRate < 0 {
		return 0.001
	}
	return errorRate
}

func (rm *RiskManager) getSystemLatency() float64 {
	// 无实时链路指标仓储时采用稳健兜底。
	return 50
}

func (rm *RiskManager) getAssessmentValue(assessment *RiskAssessment, metric RiskMetric) float64 {
	switch metric {
	case MetricVaR:
		return assessment.VaR
	case MetricCVaR:
		return assessment.CVaR
	case MetricStressTest:
		return assessment.StressLoss
	default:
		return 0
	}
}

func (rm *RiskManager) calculateBreachSeverity(value float64, limit *RiskLimit) string {
	// 计算突破程度
	breachRatio := (value - limit.LimitValue) / limit.LimitValue

	if breachRatio > 0.5 {
		return "CRITICAL"
	} else if breachRatio > 0.2 {
		return "HIGH"
	} else if breachRatio > 0.1 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

func (rm *RiskManager) calculateBreachImpact(value float64, limit *RiskLimit) float64 {
	// 估算突破影响
	breachAmount := value - limit.LimitValue

	if limit.HardLimit {
		return breachAmount * 2 // 硬限额突破影响更大
	} else {
		return breachAmount
	}
}

func (rm *RiskManager) calculateBreachProbability(limit *RiskLimit) float64 {
	base := 0.03
	if limit.HardLimit {
		base -= 0.01
	} else {
		base += 0.02
	}

	if limit.LimitValue > 0 && limit.WarningThreshold > 0 {
		warningRatio := limit.WarningThreshold / limit.LimitValue
		if warningRatio < 1 {
			base += (1 - warningRatio) * 0.1
		}
	}

	if limit.TimeWindow > 0 {
		if limit.TimeWindow <= time.Hour {
			base += 0.03
		} else if limit.TimeWindow <= 24*time.Hour {
			base += 0.01
		}
	}

	if base < 0.001 {
		return 0.001
	}
	if base > 0.5 {
		return 0.5
	}
	return base
}

// Helper functions

func generateEventID() string {
	return fmt.Sprintf("RISK_EVENT_%d", time.Now().UnixNano())
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

func getDefaultLimitValue(limitName string) float64 {
	// 默认风险限额值
	defaultLimits := map[string]float64{
		"MARKET_VOLATILITY":          0.05,    // 5%波动率
		"MARKET_SPREAD":              0.02,    // 2%价差
		"MARKET_VOLUME_RATIO":        3.0,     // 3倍平均交易量
		"CREDIT_RATING_MIN":          6.0,     // 最低信用评分
		"CREDIT_EXPOSURE_MAX":        1000000, // 100万敞口限额
		"CREDIT_CONCENTRATION_MAX":   0.2,     // 20%集中度限额
		"LIQUIDITY_ORDER_SIZE_RATIO": 0.1,     // 10%市场深度
		"LIQUIDITY_SPREAD_MAX":       0.05,    // 5%买卖价差
		"LIQUIDITY_IMPACT_MAX":       0.02,    // 2%市场冲击成本
		"OPERATIONAL_FREQUENCY_MAX":  100,     // 每分钟100笔交易
		"OPERATIONAL_ERROR_MAX":      0.01,    // 1%错误率
		"OPERATIONAL_LATENCY_MAX":    100,     // 100毫秒延迟
	}

	if value, exists := defaultLimits[limitName]; exists {
		return value
	}

	return 0
}

func calculatePortfolioValue(portfolio *Portfolio) float64 {
	total := 0.0
	for _, position := range portfolio.Positions {
		if position.MarketValue > 0 {
			total += position.MarketValue
		} else {
			total += position.Quantity * position.Price
		}
	}
	return total
}

func collectReturns(marketData []*MarketData) []float64 {
	var returns []float64
	for _, md := range marketData {
		for i := 1; i < len(md.HistoricalPrices); i++ {
			if md.HistoricalPrices[i-1] <= 0 || md.HistoricalPrices[i] <= 0 {
				continue
			}
			r := math.Log(md.HistoricalPrices[i] / md.HistoricalPrices[i-1])
			returns = append(returns, r)
		}
	}
	return returns
}

func calculateHistoricalVaRAndCVaR(returns []float64, confidenceLevel float64, portfolioValue float64) (float64, float64) {
	if len(returns) == 0 || portfolioValue <= 0 {
		return 0, 0
	}

	sorted := append([]float64(nil), returns...)
	sort.Float64s(sorted)

	idx := int((1 - confidenceLevel) * float64(len(sorted)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	varLoss := math.Max(0, -sorted[idx]*portfolioValue)

	tailSum := 0.0
	tailCount := 0
	for i := 0; i <= idx; i++ {
		if sorted[i] < 0 {
			tailSum += -sorted[i] * portfolioValue
			tailCount++
		}
	}
	if tailCount == 0 {
		return varLoss, varLoss
	}
	return varLoss, tailSum / float64(tailCount)
}

func computeRiskContributions(portfolio *Portfolio, totalRisk float64) map[string]float64 {
	contrib := make(map[string]float64)
	totalValue := calculatePortfolioValue(portfolio)
	if totalRisk <= 0 || totalValue <= 0 {
		return contrib
	}
	for _, pos := range portfolio.Positions {
		value := pos.MarketValue
		if value <= 0 {
			value = pos.Quantity * pos.Price
		}
		if value <= 0 {
			continue
		}
		contrib[pos.Symbol] += totalRisk * (value / totalValue)
	}
	return contrib
}

// Data structures

type Portfolio struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Symbols   []string    `json:"symbols"`
	Positions []*Position `json:"positions"`
}

type Position struct {
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	MarketValue float64 `json:"market_value"`
}

type MarketData struct {
	Symbol           string    `json:"symbol"`
	Bid              float64   `json:"bid"`
	Ask              float64   `json:"ask"`
	LastPrice        float64   `json:"last_price"`
	Volume           float64   `json:"volume"`
	AverageVolume    float64   `json:"average_volume"`
	HistoricalPrices []float64 `json:"historical_prices"`
	Timestamp        time.Time `json:"timestamp"`
}

type RiskCheckResult struct {
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Passed    bool      `json:"passed"`
	Reasons   []string  `json:"reasons"`
}

type LimitBreach struct {
	LimitID     string                 `json:"limit_id"`
	LimitName   string                 `json:"limit_name"`
	Metric      string                 `json:"metric"`
	Value       float64                `json:"value"`
	Limit       float64                `json:"limit"`
	Symbol      string                 `json:"symbol"`
	Severity    string                 `json:"severity"`
	Impact      float64                `json:"impact"`
	Probability float64                `json:"probability"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Interfaces

type RiskModelCalculator interface {
	Calculate(portfolio *Portfolio, marketData []*MarketData,
		riskCategory RiskCategory, metric RiskMetric) (*RiskAssessment, error)
}

type HistoricalSimulationCalculator struct{}
type ParametricCalculator struct{}
type MonteCarloCalculator struct{}
type GARCHCalculator struct{}

func (h *HistoricalSimulationCalculator) Calculate(portfolio *Portfolio, marketData []*MarketData,
	riskCategory RiskCategory, metric RiskMetric) (*RiskAssessment, error) {
	portfolioValue := calculatePortfolioValue(portfolio)
	returns := collectReturns(marketData)
	varValue, cvarValue := calculateHistoricalVaRAndCVaR(returns, 0.99, portfolioValue)

	return &RiskAssessment{
		ID:                fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
		Timestamp:         time.Now(),
		PortfolioID:       portfolio.ID,
		RiskCategory:      riskCategory,
		RiskMetric:        metric,
		RiskModel:         ModelHistoricalSimulation,
		ConfidenceLevel:   0.99,
		TimeHorizon:       24 * time.Hour,
		VaR:               varValue,
		CVaR:              cvarValue,
		ExpectedLoss:      cvarValue,
		UnexpectedLoss:    math.Max(0, varValue-cvarValue),
		RiskContributions: computeRiskContributions(portfolio, varValue),
		CorrelationMatrix: make(map[string]map[string]float64),
		Metadata: map[string]interface{}{
			"return_samples": len(returns),
		},
	}, nil
}

func (p *ParametricCalculator) Calculate(portfolio *Portfolio, marketData []*MarketData,
	riskCategory RiskCategory, metric RiskMetric) (*RiskAssessment, error) {
	portfolioValue := calculatePortfolioValue(portfolio)
	returns := collectReturns(marketData)
	if len(returns) == 0 {
		return &RiskAssessment{
			ID:              fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
			Timestamp:       time.Now(),
			PortfolioID:     portfolio.ID,
			RiskCategory:    riskCategory,
			RiskMetric:      metric,
			RiskModel:       ModelParametric,
			ConfidenceLevel: 0.99,
			TimeHorizon:     24 * time.Hour,
		}, nil
	}

	mean := calculateMean(returns)
	var variance float64
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	z := 2.326 // 99%置信度
	varValue := math.Max(0, portfolioValue*(z*stdDev-mean))
	cvarValue := math.Max(varValue, portfolioValue*(stdDev*2.665-mean))

	return &RiskAssessment{
		ID:                fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
		Timestamp:         time.Now(),
		PortfolioID:       portfolio.ID,
		RiskCategory:      riskCategory,
		RiskMetric:        metric,
		RiskModel:         ModelParametric,
		ConfidenceLevel:   0.99,
		TimeHorizon:       24 * time.Hour,
		VaR:               varValue,
		CVaR:              cvarValue,
		ExpectedLoss:      cvarValue,
		UnexpectedLoss:    math.Max(0, varValue-cvarValue),
		RiskContributions: computeRiskContributions(portfolio, varValue),
		CorrelationMatrix: make(map[string]map[string]float64),
		Metadata: map[string]interface{}{
			"return_mean": mean,
			"return_std":  stdDev,
		},
	}, nil
}

func (m *MonteCarloCalculator) Calculate(portfolio *Portfolio, marketData []*MarketData,
	riskCategory RiskCategory, metric RiskMetric) (*RiskAssessment, error) {
	portfolioValue := calculatePortfolioValue(portfolio)
	returns := collectReturns(marketData)
	if len(returns) == 0 || portfolioValue <= 0 {
		return &RiskAssessment{
			ID:              fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
			Timestamp:       time.Now(),
			PortfolioID:     portfolio.ID,
			RiskCategory:    riskCategory,
			RiskMetric:      metric,
			RiskModel:       ModelMonteCarlo,
			ConfidenceLevel: 0.99,
			TimeHorizon:     24 * time.Hour,
		}, nil
	}

	mean := calculateMean(returns)
	var variance float64
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	simCount := 2000
	simulated := make([]float64, simCount)
	for i := 0; i < simCount; i++ {
		r := rng.NormFloat64()*stdDev + mean
		simulated[i] = r
	}

	varValue, cvarValue := calculateHistoricalVaRAndCVaR(simulated, 0.99, portfolioValue)

	return &RiskAssessment{
		ID:                fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
		Timestamp:         time.Now(),
		PortfolioID:       portfolio.ID,
		RiskCategory:      riskCategory,
		RiskMetric:        metric,
		RiskModel:         ModelMonteCarlo,
		ConfidenceLevel:   0.99,
		TimeHorizon:       24 * time.Hour,
		VaR:               varValue,
		CVaR:              cvarValue,
		ExpectedLoss:      cvarValue,
		UnexpectedLoss:    math.Max(0, varValue-cvarValue),
		RiskContributions: computeRiskContributions(portfolio, varValue),
		CorrelationMatrix: make(map[string]map[string]float64),
		Metadata: map[string]interface{}{
			"sim_count": simCount,
		},
	}, nil
}

func (g *GARCHCalculator) Calculate(portfolio *Portfolio, marketData []*MarketData,
	riskCategory RiskCategory, metric RiskMetric) (*RiskAssessment, error) {
	portfolioValue := calculatePortfolioValue(portfolio)
	returns := collectReturns(marketData)
	if len(returns) == 0 || portfolioValue <= 0 {
		return &RiskAssessment{
			ID:              fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
			Timestamp:       time.Now(),
			PortfolioID:     portfolio.ID,
			RiskCategory:    riskCategory,
			RiskMetric:      metric,
			RiskModel:       ModelGARCH,
			ConfidenceLevel: 0.99,
			TimeHorizon:     24 * time.Hour,
		}, nil
	}

	omega := 0.000001
	alpha := 0.08
	beta := 0.9
	variance := math.Pow(returns[0], 2)
	for _, r := range returns[1:] {
		variance = omega + alpha*math.Pow(r, 2) + beta*variance
	}
	forecastVol := math.Sqrt(math.Max(variance, 0))

	z := 2.326
	varValue := math.Max(0, portfolioValue*z*forecastVol)
	cvarValue := math.Max(varValue, varValue*1.15)

	return &RiskAssessment{
		ID:                fmt.Sprintf("RISK_ASSESS_%d", time.Now().UnixNano()),
		Timestamp:         time.Now(),
		PortfolioID:       portfolio.ID,
		RiskCategory:      riskCategory,
		RiskMetric:        metric,
		RiskModel:         ModelGARCH,
		ConfidenceLevel:   0.99,
		TimeHorizon:       24 * time.Hour,
		VaR:               varValue,
		CVaR:              cvarValue,
		ExpectedLoss:      cvarValue,
		UnexpectedLoss:    math.Max(0, varValue-cvarValue),
		RiskContributions: computeRiskContributions(portfolio, varValue),
		CorrelationMatrix: make(map[string]map[string]float64),
		Metadata: map[string]interface{}{
			"omega": omega,
			"alpha": alpha,
			"beta":  beta,
		},
	}, nil
}

// Repository interfaces

type RiskRepository interface {
	SaveAssessment(ctx context.Context, assessment *RiskAssessment) error
	GetAssessment(ctx context.Context, id string) (*RiskAssessment, error)
	GetAssessmentsByPortfolio(ctx context.Context, portfolioID string,
		startTime, endTime time.Time) ([]*RiskAssessment, error)
	GetEventsByPortfolio(ctx context.Context, portfolioID string,
		startTime, endTime time.Time) ([]*RiskEvent, error)
	GetPortfolio(ctx context.Context, portfolioID string) (*Portfolio, error)
	GetCounterpartyRating(ctx context.Context, orderID string) (*CounterpartyRating, error)
	SaveRiskCheck(ctx context.Context, result *RiskCheckResult) error
}

type RiskLimitRepository interface {
	SaveLimit(ctx context.Context, limit *RiskLimit) error
	GetLimit(ctx context.Context, id string) (*RiskLimit, error)
	GetLimitsByCategory(ctx context.Context, category RiskCategory) ([]*RiskLimit, error)
	GetEnabledLimits(ctx context.Context) ([]*RiskLimit, error)
	UpdateLimit(ctx context.Context, limit *RiskLimit) error
	DeleteLimit(ctx context.Context, id string) error
}

type RiskEventRepository interface {
	SaveEvent(ctx context.Context, event *RiskEvent) error
	GetEvent(ctx context.Context, id string) (*RiskEvent, error)
	GetEventsByType(ctx context.Context, eventType string,
		startTime, endTime time.Time) ([]*RiskEvent, error)
	GetEventsBySeverity(ctx context.Context, severity string,
		startTime, endTime time.Time) ([]*RiskEvent, error)
	UpdateEvent(ctx context.Context, event *RiskEvent) error
}

type RiskIndicatorRepository interface {
	SaveIndicator(ctx context.Context, indicator *RiskIndicator) error
	GetIndicator(ctx context.Context, id string) (*RiskIndicator, error)
	GetIndicatorsByCategory(ctx context.Context, category RiskCategory,
		symbol string, startTime, endTime time.Time) ([]*RiskIndicator, error)
	UpdateIndicator(ctx context.Context, indicator *RiskIndicator) error
}

type MarketDataRepository interface {
	GetCurrentData(ctx context.Context, symbol string) (*MarketData, error)
	GetHistoricalData(ctx context.Context, symbols []string, days int) ([]*MarketData, error)
	GetIntradayData(ctx context.Context, symbol string, interval time.Duration,
		period time.Duration) ([]*MarketData, error)
}

type CounterpartyRating struct {
	CounterpartyID string    `json:"counterparty_id"`
	Rating         string    `json:"rating"`
	Score          float64   `json:"score"`
	ValidUntil     time.Time `json:"valid_until"`
}
