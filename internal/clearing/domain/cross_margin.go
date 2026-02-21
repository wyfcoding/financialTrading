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

// CrossMarginType 交叉保证金类型
type CrossMarginType string

const (
	CrossMarginPortfolio CrossMarginType = "PORTFOLIO" // 投资组合交叉保证金
	CrossMarginProduct   CrossMarginType = "PRODUCT"   // 产品交叉保证金
	CrossMarginAccount   CrossMarginType = "ACCOUNT"   // 账户交叉保证金
)

// CrossMarginPosition 交叉保证金持仓
type CrossMarginPosition struct {
	ID           string    `json:"id"`
	PortfolioID  string    `json:"portfolio_id"`
	MemberID     string    `json:"member_id"`
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	MarketValue  float64   `json:"market_value"`
	RiskWeight   float64   `json:"risk_weight"`
	NettingGroup string    `json:"netting_group"`
	HedgingGroup string    `json:"hedging_group"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CrossMarginGroup 交叉保证金组
type CrossMarginGroup struct {
	ID               string                 `json:"id"`
	GroupName        string                 `json:"group_name"`
	GroupType        CrossMarginType        `json:"group_type"`
	MemberID         string                 `json:"member_id"`
	Positions        []*CrossMarginPosition `json:"positions"`
	TotalMarketValue float64                `json:"total_market_value"`
	TotalRiskWeight  float64                `json:"total_risk_weight"`
	NetMargin        float64                `json:"net_margin"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// CrossMarginCalculator 交叉保证金计算器
type CrossMarginCalculator struct {
	riskModel       RiskModel
	correlationRepo CorrelationRepository
	mu              sync.RWMutex
	config          *CrossMarginConfig
}

// CrossMarginConfig 交叉保证金配置
type CrossMarginConfig struct {
	CorrelationThreshold   float64 `json:"correlation_threshold"`
	DiversificationBenefit float64 `json:"diversification_benefit"`
	HedgingBenefit         float64 `json:"hedging_benefit"`
	NettingBenefit         float64 `json:"netting_benefit"`
	MaxLeverage            float64 `json:"max_leverage"`
	MinHaircut             float64 `json:"min_haircut"`
}

// NewCrossMarginCalculator 创建交叉保证金计算器
func NewCrossMarginCalculator(riskModel RiskModel, correlationRepo CorrelationRepository) *CrossMarginCalculator {
	return &CrossMarginCalculator{
		riskModel:       riskModel,
		correlationRepo: correlationRepo,
		config: &CrossMarginConfig{
			CorrelationThreshold:   0.7,
			DiversificationBenefit: 0.3,
			HedgingBenefit:         0.5,
			NettingBenefit:         0.8,
			MaxLeverage:            10.0,
			MinHaircut:             0.05,
		},
	}
}

// CalculatePortfolioMargin 计算投资组合保证金
func (cmc *CrossMarginCalculator) CalculatePortfolioMargin(ctx context.Context, portfolio *Portfolio) (*MarginResult, error) {
	// 收集持仓数据
	positions := cmc.collectPositions(portfolio)

	// 计算风险权重
	riskWeights := cmc.calculateRiskWeights(ctx, positions)

	// 计算相关性矩阵
	correlationMatrix, err := cmc.getCorrelationMatrix(ctx, positions)
	if err != nil {
		return nil, fmt.Errorf("failed to get correlation matrix: %w", err)
	}

	// 计算投资组合保证金
	margin := cmc.calculatePortfolioMargin(positions, riskWeights, correlationMatrix)

	result := &MarginResult{
		PortfolioID:      portfolio.ID,
		TotalMarketValue: cmc.calculateTotalMarketValue(positions),
		TotalRiskWeight:  cmc.calculateTotalRiskWeight(positions, riskWeights),
		NetMargin:        margin,
		Components:       cmc.calculateMarginComponents(positions, riskWeights),
		CalculatedAt:     time.Now(),
	}

	return result, nil
}

// collectPositions 收集持仓数据
func (cmc *CrossMarginCalculator) collectPositions(portfolio *Portfolio) []*CrossMarginPosition {
	var positions []*CrossMarginPosition

	for _, position := range portfolio.Positions {
		crossPosition := &CrossMarginPosition{
			ID:          generatePositionID(),
			PortfolioID: portfolio.ID,
			MemberID:    portfolio.MemberID,
			Symbol:      position.Symbol,
			Quantity:    position.Quantity,
			Price:       position.Price,
			MarketValue: position.MarketValue,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		positions = append(positions, crossPosition)
	}

	return positions
}

// calculateRiskWeights 计算风险权重
func (cmc *CrossMarginCalculator) calculateRiskWeights(ctx context.Context, positions []*CrossMarginPosition) map[string]float64 {
	riskWeights := make(map[string]float64)

	for _, position := range positions {
		// 根据风险模型计算风险权重
		riskWeight := cmc.riskModel.CalculateRiskWeight(position.Symbol, position.MarketValue)
		riskWeights[position.ID] = riskWeight
		position.RiskWeight = riskWeight
	}

	return riskWeights
}

// getCorrelationMatrix 获取相关性矩阵
func (cmc *CrossMarginCalculator) getCorrelationMatrix(ctx context.Context, positions []*CrossMarginPosition) (map[string]map[string]float64, error) {
	matrix := make(map[string]map[string]float64)

	// 获取所有符号
	var symbols []string
	for _, position := range positions {
		symbols = append(symbols, position.Symbol)
	}

	// 获取相关性数据
	correlations, err := cmc.correlationRepo.GetCorrelations(ctx, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to get correlations: %w", err)
	}

	// 构建矩阵
	for i, pos1 := range positions {
		matrix[pos1.ID] = make(map[string]float64)

		for j, pos2 := range positions {
			if i == j {
				matrix[pos1.ID][pos2.ID] = 1.0
			} else {
				// 查找相关性
				corr := cmc.findCorrelation(correlations, pos1.Symbol, pos2.Symbol)
				matrix[pos1.ID][pos2.ID] = corr
			}
		}
	}

	return matrix, nil
}

// findCorrelation 查找相关性
func (cmc *CrossMarginCalculator) findCorrelation(correlations []*Correlation, symbol1, symbol2 string) float64 {
	for _, corr := range correlations {
		if (corr.Symbol1 == symbol1 && corr.Symbol2 == symbol2) ||
			(corr.Symbol1 == symbol2 && corr.Symbol2 == symbol1) {
			return corr.Correlation
		}
	}

	// 默认相关性
	return 0.3
}

// calculatePortfolioMargin 计算投资组合保证金
func (cmc *CrossMarginCalculator) calculatePortfolioMargin(positions []*CrossMarginPosition,
	riskWeights map[string]float64, correlationMatrix map[string]map[string]float64) float64 {

	if len(positions) == 0 {
		return 0
	}

	if len(positions) == 1 {
		// 单一持仓，无交叉保证金优惠
		position := positions[0]
		return position.MarketValue * riskWeights[position.ID]
	}

	// 计算投资组合方差
	var portfolioVariance float64

	for _, pos1 := range positions {
		for _, pos2 := range positions {
			weight1 := pos1.MarketValue * riskWeights[pos1.ID]
			weight2 := pos2.MarketValue * riskWeights[pos2.ID]
			corr := correlationMatrix[pos1.ID][pos2.ID]

			portfolioVariance += weight1 * weight2 * corr
		}
	}

	// 计算投资组合标准差
	portfolioStdDev := math.Sqrt(math.Abs(portfolioVariance))

	// 应用多样化收益
	diversifiedMargin := portfolioStdDev * (1 - cmc.config.DiversificationBenefit)

	// 应用对冲收益
	hedgingBenefit := cmc.calculateHedgingBenefit(positions)
	hedgedMargin := diversifiedMargin * (1 - hedgingBenefit)

	// 应用净额收益
	nettingBenefit := cmc.calculateNettingBenefit(positions)
	finalMargin := hedgedMargin * (1 - nettingBenefit)

	// 确保保证金不低于最低要求
	minMargin := cmc.calculateMinMargin(positions)
	if finalMargin < minMargin {
		finalMargin = minMargin
	}

	return finalMargin
}

// calculateHedgingBenefit 计算对冲收益
func (cmc *CrossMarginCalculator) calculateHedgingBenefit(positions []*CrossMarginPosition) float64 {
	// 识别对冲关系
	hedgingPairs := cmc.identifyHedgingPairs(positions)

	if len(hedgingPairs) == 0 {
		return 0
	}

	// 计算对冲收益
	totalHedgingBenefit := 0.0
	for _, pair := range hedgingPairs {
		// 对冲程度越高，收益越大
		hedgingRatio := math.Min(pair.Position1.MarketValue, pair.Position2.MarketValue) /
			math.Max(pair.Position1.MarketValue, pair.Position2.MarketValue)
		totalHedgingBenefit += hedgingRatio * cmc.config.HedgingBenefit
	}

	// 平均对冲收益
	avgHedgingBenefit := totalHedgingBenefit / float64(len(hedgingPairs))

	return math.Min(avgHedgingBenefit, cmc.config.HedgingBenefit)
}

// calculateNettingBenefit 计算净额收益
func (cmc *CrossMarginCalculator) calculateNettingBenefit(positions []*CrossMarginPosition) float64 {
	// 识别净额关系
	nettingGroups := cmc.groupByNetting(positions)

	if len(nettingGroups) == 0 {
		return 0
	}

	// 计算净额收益
	totalNettingBenefit := 0.0
	for _, group := range nettingGroups {
		// 净额程度越高，收益越大
		nettingRatio := cmc.calculateNettingRatio(group)
		totalNettingBenefit += nettingRatio * cmc.config.NettingBenefit
	}

	// 平均净额收益
	avgNettingBenefit := totalNettingBenefit / float64(len(nettingGroups))

	return math.Min(avgNettingBenefit, cmc.config.NettingBenefit)
}

// calculateMinMargin 计算最低保证金
func (cmc *CrossMarginCalculator) calculateMinMargin(positions []*CrossMarginPosition) float64 {
	var totalMarketValue float64
	for _, position := range positions {
		totalMarketValue += position.MarketValue
	}

	// 最低保证金 = 总市值 * 最低折扣率
	minMargin := totalMarketValue * cmc.config.MinHaircut

	return minMargin
}

// calculateTotalMarketValue 计算总市值
func (cmc *CrossMarginCalculator) calculateTotalMarketValue(positions []*CrossMarginPosition) float64 {
	var total float64
	for _, position := range positions {
		total += position.MarketValue
	}
	return total
}

// calculateTotalRiskWeight 计算总风险权重
func (cmc *CrossMarginCalculator) calculateTotalRiskWeight(positions []*CrossMarginPosition, riskWeights map[string]float64) float64 {
	var total float64
	for _, position := range positions {
		total += position.MarketValue * riskWeights[position.ID]
	}
	return total
}

// calculateMarginComponents 计算保证金组件
func (cmc *CrossMarginCalculator) calculateMarginComponents(positions []*CrossMarginPosition, riskWeights map[string]float64) []*MarginComponent {
	var components []*MarginComponent

	for _, position := range positions {
		component := &MarginComponent{
			PositionID:        position.ID,
			Symbol:            position.Symbol,
			MarketValue:       position.MarketValue,
			RiskWeight:        riskWeights[position.ID],
			MarginRequirement: position.MarketValue * riskWeights[position.ID],
		}
		components = append(components, component)
	}

	return components
}

// identifyHedgingPairs 识别对冲对
func (cmc *CrossMarginCalculator) identifyHedgingPairs(positions []*CrossMarginPosition) []*HedgingPair {
	var pairs []*HedgingPair

	// 简化实现：识别相反方向的相同符号持仓
	positionMap := make(map[string]*CrossMarginPosition)

	for _, position := range positions {
		if existing, exists := positionMap[position.Symbol]; exists {
			// 检查是否为对冲关系
			if existing.Quantity*position.Quantity < 0 {
				pair := &HedgingPair{
					Position1: existing,
					Position2: position,
				}
				pairs = append(pairs, pair)
			}
		} else {
			positionMap[position.Symbol] = position
		}
	}

	return pairs
}

// groupByNetting 按净额分组
func (cmc *CrossMarginCalculator) groupByNetting(positions []*CrossMarginPosition) map[string][]*CrossMarginPosition {
	groups := make(map[string][]*CrossMarginPosition)

	for _, position := range positions {
		groupKey := position.NettingGroup
		if groupKey == "" {
			groupKey = "default"
		}

		groups[groupKey] = append(groups[groupKey], position)
	}

	return groups
}

// calculateNettingRatio 计算净额比率
func (cmc *CrossMarginCalculator) calculateNettingRatio(group []*CrossMarginPosition) float64 {
	if len(group) == 0 {
		return 0
	}

	// 计算净头寸
	var netQuantity float64
	var grossQuantity float64

	for _, position := range group {
		netQuantity += position.Quantity
		grossQuantity += math.Abs(position.Quantity)
	}

	if grossQuantity == 0 {
		return 0
	}

	// 净额比率 = 净头寸 / 总头寸
	nettingRatio := math.Abs(netQuantity) / grossQuantity

	return nettingRatio
}

// Data structures

type MarginResult struct {
	PortfolioID      string             `json:"portfolio_id"`
	TotalMarketValue float64            `json:"total_market_value"`
	TotalRiskWeight  float64            `json:"total_risk_weight"`
	NetMargin        float64            `json:"net_margin"`
	Components       []*MarginComponent `json:"components"`
	CalculatedAt     time.Time          `json:"calculated_at"`
}

type MarginComponent struct {
	PositionID        string  `json:"position_id"`
	Symbol            string  `json:"symbol"`
	MarketValue       float64 `json:"market_value"`
	RiskWeight        float64 `json:"risk_weight"`
	MarginRequirement float64 `json:"margin_requirement"`
}

type HedgingPair struct {
	Position1 *CrossMarginPosition `json:"position1"`
	Position2 *CrossMarginPosition `json:"position2"`
}

type Correlation struct {
	Symbol1      string    `json:"symbol1"`
	Symbol2      string    `json:"symbol2"`
	Correlation  float64   `json:"correlation"`
	Period       string    `json:"period"` // DAILY, WEEKLY, MONTHLY
	CalculatedAt time.Time `json:"calculated_at"`
}

// Repository interfaces

type CorrelationRepository interface {
	GetCorrelation(ctx context.Context, symbol1, symbol2 string) (*Correlation, error)
	GetCorrelations(ctx context.Context, symbols []string) ([]*Correlation, error)
	SaveCorrelation(ctx context.Context, correlation *Correlation) error
	UpdateCorrelation(ctx context.Context, correlation *Correlation) error
}

// Helper functions

func generatePositionID() string {
	return fmt.Sprintf("POS_%d", time.Now().UnixNano())
}
