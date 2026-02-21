//go:build risk_experimental
// +build risk_experimental

package domain

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// BacktestType 回测类型
type BacktestType string

const (
	BacktestKupiec         BacktestType = "KUPIEC"         // Kupiec检验
	BacktestChristoffersen BacktestType = "CHRISTOFFERSEN" // Christoffersen检验
	BacktestMixed          BacktestType = "MIXED"          // 混合检验
)

// BacktestResult 回测结果
type BacktestResult struct {
	ID              string        `json:"id"`
	BacktestType    BacktestType  `json:"backtest_type"`
	PortfolioID     string        `json:"portfolio_id"`
	RiskModel       RiskModel     `json:"risk_model"`
	ConfidenceLevel float64       `json:"confidence_level"`
	TimeHorizon     time.Duration `json:"time_horizon"`

	// 回测统计
	TotalDays        int     `json:"total_days"`
	VaRBreaches      int     `json:"var_breaches"`
	ExpectedBreaches float64 `json:"expected_breaches"`
	BreachRate       float64 `json:"breach_rate"`
	ExpectedRate     float64 `json:"expected_rate"`

	// 检验统计量
	TestStatistic float64 `json:"test_statistic"`
	CriticalValue float64 `json:"critical_value"`
	PValue        float64 `json:"p_value"`

	// 检验结果
	Passed     bool   `json:"passed"`
	Conclusion string `json:"conclusion"`
	Confidence string `json:"confidence"` // HIGH, MEDIUM, LOW

	// 时间序列
	VaRSeries    []float64 `json:"var_series"`
	PnlSeries    []float64 `json:"pnl_series"`
	BreachSeries []bool    `json:"breach_series"`

	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// KupiecTest Kupiec检验实现
type KupiecTest struct {
	confidenceLevel float64
	alpha           float64 // 显著性水平
}

// NewKupiecTest 创建Kupiec检验
func NewKupiecTest(confidenceLevel, alpha float64) *KupiecTest {
	return &KupiecTest{
		confidenceLevel: confidenceLevel,
		alpha:           alpha,
	}
}

// Test 执行Kupiec检验
func (kt *KupiecTest) Test(backtest *BacktestResult) *BacktestResult {
	// 计算期望突破次数
	expectedBreaches := float64(backtest.TotalDays) * (1 - kt.confidenceLevel)
	backtest.ExpectedBreaches = expectedBreaches

	// 计算实际突破率
	breachRate := float64(backtest.VaRBreaches) / float64(backtest.TotalDays)
	backtest.BreachRate = breachRate
	backtest.ExpectedRate = 1 - kt.confidenceLevel

	// 计算似然比统计量
	LR := kt.calculateLikelihoodRatio(backtest.VaRBreaches, backtest.TotalDays, kt.confidenceLevel)
	backtest.TestStatistic = LR

	// 计算临界值（卡方分布，自由度为1）
	criticalValue := kt.calculateCriticalValue(kt.alpha)
	backtest.CriticalValue = criticalValue

	// 计算p值
	pValue := kt.calculatePValue(LR)
	backtest.PValue = pValue

	// 判断是否通过检验
	backtest.Passed = LR <= criticalValue

	// 生成结论
	backtest.Conclusion = kt.generateConclusion(backtest.Passed, breachRate, 1-kt.confidenceLevel)

	// 确定置信度
	backtest.Confidence = kt.determineConfidence(pValue)

	return backtest
}

// calculateLikelihoodRatio 计算似然比统计量
func (kt *KupiecTest) calculateLikelihoodRatio(x, n int, p float64) float64 {
	if x == 0 {
		x = 1 // 避免log(0)
	}
	if x == n {
		x = n - 1 // 避免log(0)
	}

	// 似然函数
	L0 := math.Pow(p, float64(x)) * math.Pow(1-p, float64(n-x))
	L1 := math.Pow(float64(x)/float64(n), float64(x)) * math.Pow(1-float64(x)/float64(n), float64(n-x))

	// 似然比统计量
	LR := -2 * math.Log(L0/L1)

	return LR
}

// calculateCriticalValue 计算临界值
func (kt *KupiecTest) calculateCriticalValue(alpha float64) float64 {
	// 卡方分布临界值（自由度为1）
	// 简化实现，实际应该使用统计库
	criticalValues := map[float64]float64{
		0.01: 6.635, // 99%置信度
		0.05: 3.841, // 95%置信度
		0.10: 2.706, // 90%置信度
	}

	if cv, exists := criticalValues[alpha]; exists {
		return cv
	}

	// 默认值
	return 3.841
}

// calculatePValue 计算p值
func (kt *KupiecTest) calculatePValue(LR float64) float64 {
	// 简化实现，实际应该使用统计库
	// 基于卡方分布计算p值
	if LR < 2.706 {
		return 0.10
	} else if LR < 3.841 {
		return 0.05
	} else if LR < 6.635 {
		return 0.01
	} else {
		return 0.001
	}
}

// generateConclusion 生成结论
func (kt *KupiecTest) generateConclusion(passed bool, actualRate, expectedRate float64) string {
	if passed {
		return fmt.Sprintf("模型通过检验，实际突破率%.4f接近期望突破率%.4f", actualRate, expectedRate)
	} else {
		return fmt.Sprintf("模型未通过检验，实际突破率%.4f显著偏离期望突破率%.4f", actualRate, expectedRate)
	}
}

// determineConfidence 确定置信度
func (kt *KupiecTest) determineConfidence(pValue float64) string {
	if pValue < 0.01 {
		return "HIGH"
	} else if pValue < 0.05 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

// ChristoffersenTest Christoffersen检验实现
type ChristoffersenTest struct {
	confidenceLevel float64
	alpha           float64
}

// NewChristoffersenTest 创建Christoffersen检验
func NewChristoffersenTest(confidenceLevel, alpha float64) *ChristoffersenTest {
	return &ChristoffersenTest{
		confidenceLevel: confidenceLevel,
		alpha:           alpha,
	}
}

// Test 执行Christoffersen检验
func (ct *ChristoffersenTest) Test(backtest *BacktestResult) *BacktestResult {
	// 计算条件覆盖检验统计量
	LRcc := ct.calculateConditionalCoverageLR(backtest.BreachSeries)
	backtest.TestStatistic = LRcc

	// 计算临界值（卡方分布，自由度为2）
	criticalValue := ct.calculateCriticalValue(ct.alpha)
	backtest.CriticalValue = criticalValue

	// 计算p值
	pValue := ct.calculatePValue(LRcc)
	backtest.PValue = pValue

	// 判断是否通过检验
	backtest.Passed = LRcc <= criticalValue

	// 生成结论
	backtest.Conclusion = ct.generateConclusion(backtest.Passed, backtest.BreachSeries)

	return backtest
}

// calculateConditionalCoverageLR 计算条件覆盖似然比统计量
func (ct *ChristoffersenTest) calculateConditionalCoverageLR(breachSeries []bool) float64 {
	if len(breachSeries) < 2 {
		return 0
	}

	// 计算转移矩阵
	n00, n01, n10, n11 := 0, 0, 0, 0

	for i := 1; i < len(breachSeries); i++ {
		prev := breachSeries[i-1]
		curr := breachSeries[i]

		if !prev && !curr {
			n00++
		} else if !prev && curr {
			n01++
		} else if prev && !curr {
			n10++
		} else {
			n11++
		}
	}

	// 计算转移概率
	pi01 := float64(n01) / float64(n00+n01)
	pi11 := float64(n11) / float64(n10+n11)
	pi := float64(n01+n11) / float64(n00+n01+n10+n11)

	// 计算似然函数
	L0 := math.Pow(1-pi, float64(n00+n10)) * math.Pow(pi, float64(n01+n11))
	L1 := math.Pow(1-pi01, float64(n00)) * math.Pow(pi01, float64(n01)) *
		math.Pow(1-pi11, float64(n10)) * math.Pow(pi11, float64(n11))

	// 似然比统计量
	LRcc := -2 * math.Log(L0/L1)

	return LRcc
}

// calculateCriticalValue 计算临界值
func (ct *ChristoffersenTest) calculateCriticalValue(alpha float64) float64 {
	// 卡方分布临界值（自由度为2）
	criticalValues := map[float64]float64{
		0.01: 9.210, // 99%置信度
		0.05: 5.991, // 95%置信度
		0.10: 4.605, // 90%置信度
	}

	if cv, exists := criticalValues[alpha]; exists {
		return cv
	}

	// 默认值
	return 5.991
}

// calculatePValue 计算p值
func (ct *ChristoffersenTest) calculatePValue(LR float64) float64 {
	// 简化实现
	if LR < 4.605 {
		return 0.10
	} else if LR < 5.991 {
		return 0.05
	} else if LR < 9.210 {
		return 0.01
	} else {
		return 0.001
	}
}

// generateConclusion 生成结论
func (ct *ChristoffersenTest) generateConclusion(passed bool, breachSeries []bool) string {
	if passed {
		return "模型通过条件覆盖检验，突破事件独立"
	} else {
		// 分析突破模式
		clustering := ct.analyzeClustering(breachSeries)
		return fmt.Sprintf("模型未通过条件覆盖检验，突破事件存在%s", clustering)
	}
}

// analyzeClustering 分析突破聚类
func (ct *ChristoffersenTest) analyzeClustering(breachSeries []bool) string {
	// 计算连续突破次数
	maxConsecutive := 0
	current := 0

	for _, breach := range breachSeries {
		if breach {
			current++
			if current > maxConsecutive {
				maxConsecutive = current
			}
		} else {
			current = 0
		}
	}

	if maxConsecutive > 3 {
		return fmt.Sprintf("明显聚类（最大连续突破%d次）", maxConsecutive)
	} else {
		return "轻微聚类现象"
	}
}

// RiskLimitCascade 风险限额级联
type RiskLimitCascade struct {
	Levels []*LimitLevel
	mu     sync.RWMutex
}

// LimitLevel 限额层级
type LimitLevel struct {
	LevelName  string             `json:"level_name"`
	EntityType string             `json:"entity_type"` // ACCOUNT, PORTFOLIO, FIRM
	EntityID   string             `json:"entity_id"`
	ParentID   string             `json:"parent_id"`
	Limits     map[string]float64 `json:"limits"` // 限额类型->值
	Usage      map[string]float64 `json:"usage"`  // 限额类型->使用量
	Children   []*LimitLevel      `json:"children"`
}

// NewRiskLimitCascade 创建风险限额级联
func NewRiskLimitCascade() *RiskLimitCascade {
	return &RiskLimitCascade{
		Levels: make([]*LimitLevel, 0),
	}
}

// AddLevel 添加限额层级
func (rlc *RiskLimitCascade) AddLevel(level *LimitLevel) {
	rlc.mu.Lock()
	defer rlc.mu.Unlock()

	rlc.Levels = append(rlc.Levels, level)
}

// CheckLimit 检查限额
func (rlc *RiskLimitCascade) CheckLimit(entityID, limitType string, value float64) (bool, []string) {
	rlc.mu.RLock()
	defer rlc.mu.RUnlock()

	var violations []string

	// 查找实体
	entity := rlc.findEntity(entityID)
	if entity == nil {
		return false, []string{"Entity not found"}
	}

	// 检查本级限额
	if limit, exists := entity.Limits[limitType]; exists {
		currentUsage := entity.Usage[limitType]
		if currentUsage+value > limit {
			violations = append(violations,
				fmt.Sprintf("Entity %s %s limit exceeded: %.2f/%.2f",
					entityID, limitType, currentUsage+value, limit))
		}
	}

	// 递归检查父级限额
	rlc.checkParentLimits(entity, limitType, value, &violations)

	return len(violations) == 0, violations
}

// findEntity 查找实体
func (rlc *RiskLimitCascade) findEntity(entityID string) *LimitLevel {
	for _, level := range rlc.Levels {
		if found := rlc.findEntityDFS(level, entityID); found != nil {
			return found
		}
	}
	return nil
}

// findEntityDFS 深度优先搜索实体
func (rlc *RiskLimitCascade) findEntityDFS(node *LimitLevel, entityID string) *LimitLevel {
	if node.EntityID == entityID {
		return node
	}

	for _, child := range node.Children {
		if found := rlc.findEntityDFS(child, entityID); found != nil {
			return found
		}
	}

	return nil
}

// checkParentLimits 检查父级限额
func (rlc *RiskLimitCascade) checkParentLimits(entity *LimitLevel, limitType string, value float64, violations *[]string) {
	if entity.ParentID == "" {
		return
	}

	parent := rlc.findEntity(entity.ParentID)
	if parent == nil {
		return
	}

	// 检查父级限额
	if limit, exists := parent.Limits[limitType]; exists {
		currentUsage := parent.Usage[limitType]
		if currentUsage+value > limit {
			*violations = append(*violations,
				fmt.Sprintf("Parent %s %s limit exceeded: %.2f/%.2f",
					parent.EntityID, limitType, currentUsage+value, limit))
		}
	}

	// 递归检查更高级别
	rlc.checkParentLimits(parent, limitType, value, violations)
}

// UpdateUsage 更新限额使用量
func (rlc *RiskLimitCascade) UpdateUsage(entityID, limitType string, value float64) error {
	rlc.mu.Lock()
	defer rlc.mu.Unlock()

	entity := rlc.findEntity(entityID)
	if entity == nil {
		return fmt.Errorf("entity not found: %s", entityID)
	}

	// 更新本级使用量
	if _, exists := entity.Usage[limitType]; !exists {
		entity.Usage[limitType] = 0
	}
	entity.Usage[limitType] += value

	// 递归更新父级使用量
	rlc.updateParentUsage(entity, limitType, value)

	return nil
}

// updateParentUsage 更新父级使用量
func (rlc *RiskLimitCascade) updateParentUsage(entity *LimitLevel, limitType string, value float64) {
	if entity.ParentID == "" {
		return
	}

	parent := rlc.findEntity(entity.ParentID)
	if parent == nil {
		return
	}

	// 更新父级使用量
	if _, exists := parent.Usage[limitType]; !exists {
		parent.Usage[limitType] = 0
	}
	parent.Usage[limitType] += value

	// 递归更新更高级别
	rlc.updateParentUsage(parent, limitType, value)
}

// RealTimePnlCalculator 实时P&L计算器
type RealTimePnlCalculator struct {
	positions    map[string]*PositionPnl
	marketData   map[string]*MarketPrice
	riskFreeRate float64
	mu           sync.RWMutex
}

// PositionPnl 持仓P&L
type PositionPnl struct {
	PositionID    string    `json:"position_id"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AvgCost       float64   `json:"avg_cost"`
	MarketPrice   float64   `json:"market_price"`
	UnrealizedPnl float64   `json:"unrealized_pnl"`
	RealizedPnl   float64   `json:"realized_pnl"`
	TotalPnl      float64   `json:"total_pnl"`
	PnlPercent    float64   `json:"pnl_percent"`
	Timestamp     time.Time `json:"timestamp"`
}

// MarketPrice 市场价格
type MarketPrice struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Last      float64   `json:"last"`
	Timestamp time.Time `json:"timestamp"`
}

// NewRealTimePnlCalculator 创建实时P&L计算器
func NewRealTimePnlCalculator(riskFreeRate float64) *RealTimePnlCalculator {
	return &RealTimePnlCalculator{
		positions:    make(map[string]*PositionPnl),
		marketData:   make(map[string]*MarketPrice),
		riskFreeRate: riskFreeRate,
	}
}

// UpdatePosition 更新持仓
func (rtpc *RealTimePnlCalculator) UpdatePosition(positionID, symbol string, quantity, avgCost float64) {
	rtpc.mu.Lock()
	defer rtpc.mu.Unlock()

	position := &PositionPnl{
		PositionID: positionID,
		Symbol:     symbol,
		Quantity:   quantity,
		AvgCost:    avgCost,
		Timestamp:  time.Now(),
	}

	rtpc.positions[positionID] = position

	// 如果有市场价格，计算P&L
	if marketPrice, exists := rtpc.marketData[symbol]; exists {
		rtpc.calculatePnl(position, marketPrice)
	}
}

// UpdateMarketPrice 更新市场价格
func (rtpc *RealTimePnlCalculator) UpdateMarketPrice(symbol string, bid, ask, last float64) {
	rtpc.mu.Lock()
	defer rtpc.mu.Unlock()

	marketPrice := &MarketPrice{
		Symbol:    symbol,
		Bid:       bid,
		Ask:       ask,
		Last:      last,
		Timestamp: time.Now(),
	}

	rtpc.marketData[symbol] = marketPrice

	// 更新所有相关持仓的P&L
	for _, position := range rtpc.positions {
		if position.Symbol == symbol {
			rtpc.calculatePnl(position, marketPrice)
		}
	}
}

// calculatePnl 计算P&L
func (rtpc *RealTimePnlCalculator) calculatePnl(position *PositionPnl, marketPrice *MarketPrice) {
	// 使用中间价作为市场价格
	marketValue := (marketPrice.Bid + marketPrice.Ask) / 2
	if marketValue <= 0 {
		marketValue = marketPrice.Last
	}

	position.MarketPrice = marketValue

	// 计算未实现P&L
	costValue := position.Quantity * position.AvgCost
	currentValue := position.Quantity * marketValue
	position.UnrealizedPnl = currentValue - costValue

	// 计算总P&L
	position.TotalPnl = position.UnrealizedPnl + position.RealizedPnl

	// 计算P&L百分比
	if costValue > 0 {
		position.PnlPercent = position.UnrealizedPnl / costValue * 100
	}

	position.Timestamp = time.Now()
}

// RecordRealizedPnl 记录已实现P&L
func (rtpc *RealTimePnlCalculator) RecordRealizedPnl(positionID string, realizedPnl float64) {
	rtpc.mu.Lock()
	defer rtpc.mu.Unlock()

	if position, exists := rtpc.positions[positionID]; exists {
		position.RealizedPnl += realizedPnl
		position.TotalPnl = position.UnrealizedPnl + position.RealizedPnl
		position.Timestamp = time.Now()
	}
}

// GetPositionPnl 获取持仓P&L
func (rtpc *RealTimePnlCalculator) GetPositionPnl(positionID string) (*PositionPnl, error) {
	rtpc.mu.RLock()
	defer rtpc.mu.RUnlock()

	position, exists := rtpc.positions[positionID]
	if !exists {
		return nil, fmt.Errorf("position not found: %s", positionID)
	}

	return position, nil
}

// GetPortfolioPnl 获取投资组合P&L
func (rtpc *RealTimePnlCalculator) GetPortfolioPnl() *PortfolioPnl {
	rtpc.mu.RLock()
	defer rtpc.mu.RUnlock()

	portfolio := &PortfolioPnl{
		Timestamp:          time.Now(),
		Positions:          make([]*PositionPnl, 0),
		TotalCost:          0,
		TotalMarketValue:   0,
		TotalUnrealizedPnl: 0,
		TotalRealizedPnl:   0,
		TotalPnl:           0,
	}

	for _, position := range rtpc.positions {
		portfolio.Positions = append(portfolio.Positions, position)
		portfolio.TotalCost += position.Quantity * position.AvgCost
		portfolio.TotalMarketValue += position.Quantity * position.MarketPrice
		portfolio.TotalUnrealizedPnl += position.UnrealizedPnl
		portfolio.TotalRealizedPnl += position.RealizedPnl
		portfolio.TotalPnl += position.TotalPnl
	}

	// 计算投资组合收益率
	if portfolio.TotalCost > 0 {
		portfolio.PnlPercent = portfolio.TotalPnl / portfolio.TotalCost * 100
	}

	return portfolio
}

// PortfolioPnl 投资组合P&L
type PortfolioPnl struct {
	Timestamp          time.Time      `json:"timestamp"`
	Positions          []*PositionPnl `json:"positions"`
	TotalCost          float64        `json:"total_cost"`
	TotalMarketValue   float64        `json:"total_market_value"`
	TotalUnrealizedPnl float64        `json:"total_unrealized_pnl"`
	TotalRealizedPnl   float64        `json:"total_realized_pnl"`
	TotalPnl           float64        `json:"total_pnl"`
	PnlPercent         float64        `json:"pnl_percent"`
}

// TailRiskMetrics 尾部风险度量
type TailRiskMetrics struct {
	CVaR              float64 `json:"cvar"`               // 条件风险价值
	ExpectedShortfall float64 `json:"expected_shortfall"` // 期望损失
	TailVaR           float64 `json:"tail_var"`           // 尾部VaR
	TailIndex         float64 `json:"tail_index"`         // 尾部指数
	MaxDrawdown       float64 `json:"max_drawdown"`       // 最大回撤
	StressLoss        float64 `json:"stress_loss"`        // 压力损失
}

// CalculateTailRisk 计算尾部风险
func CalculateTailRisk(returns []float64, confidenceLevel float64) *TailRiskMetrics {
	if len(returns) == 0 {
		return &TailRiskMetrics{}
	}

	// 排序收益率
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sortFloat64s(sortedReturns)

	// 计算VaR
	varIndex := int(float64(len(sortedReturns)) * (1 - confidenceLevel))
	if varIndex < 0 {
		varIndex = 0
	}
	if varIndex >= len(sortedReturns) {
		varIndex = len(sortedReturns) - 1
	}
	VaR := -sortedReturns[varIndex] // 取负值，表示损失

	// 计算CVaR/ES
	var tailReturns []float64
	for _, ret := range sortedReturns {
		if -ret >= VaR { // 损失大于等于VaR
			tailReturns = append(tailReturns, -ret)
		}
	}

	var cvar float64
	if len(tailReturns) > 0 {
		for _, loss := range tailReturns {
			cvar += loss
		}
		cvar /= float64(len(tailReturns))
	}

	// 计算尾部指数
	tailIndex := calculateTailIndex(sortedReturns)

	// 计算最大回撤
	maxDrawdown := calculateMaxDrawdown(returns)

	return &TailRiskMetrics{
		CVaR:              cvar,
		ExpectedShortfall: cvar,
		TailVaR:           VaR,
		TailIndex:         tailIndex,
		MaxDrawdown:       maxDrawdown,
		StressLoss:        cvar * 2, // 简化估计
	}
}

// calculateTailIndex 计算尾部指数
func calculateTailIndex(returns []float64) float64 {
	// Hill估计器简化实现
	if len(returns) < 10 {
		return 3.0 // 默认值
	}

	// 取最大的10%收益率
	k := len(returns) / 10
	if k < 5 {
		k = 5
	}

	var sumLogRatio float64
	for i := 0; i < k; i++ {
		sumLogRatio += math.Log(returns[len(returns)-1-i] / returns[len(returns)-k-1])
	}

	tailIndex := 1.0 / (sumLogRatio / float64(k))

	return tailIndex
}

// calculateMaxDrawdown 计算最大回撤
func calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 计算累积收益
	cumulative := make([]float64, len(returns))
	cumulative[0] = 1 + returns[0]
	for i := 1; i < len(returns); i++ {
		cumulative[i] = cumulative[i-1] * (1 + returns[i])
	}

	// 计算最大回撤
	peak := cumulative[0]
	maxDrawdown := 0.0

	for i := 1; i < len(cumulative); i++ {
		if cumulative[i] > peak {
			peak = cumulative[i]
		}

		drawdown := (peak - cumulative[i]) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// Helper function for sorting
func sortFloat64s(arr []float64) {
	// 简单冒泡排序
	n := len(arr)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
}
