// 变更说明：完善投资组合领域模型，增加组合风险归因、Brinson业绩归因、资产配置优化等高级功能
package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Portfolio 投资组合聚合根
type Portfolio struct {
	gorm.Model
	PortfolioID     string          `gorm:"column:portfolio_id;type:varchar(32);uniqueIndex;not null" json:"portfolio_id"`
	UserID          string          `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	Name            string          `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Description     string          `gorm:"column:description;type:text" json:"description"`
	BaseCurrency    string          `gorm:"column:base_currency;type:varchar(3);not null;default:'USD'" json:"base_currency"`
	TotalValue      decimal.Decimal `gorm:"column:total_value;type:decimal(20,4);not null;default:0" json:"total_value"`
	CashValue       decimal.Decimal `gorm:"column:cash_value;type:decimal(20,4);not null;default:0" json:"cash_value"`
	PositionsValue  decimal.Decimal `gorm:"column:positions_value;type:decimal(20,4);not null;default:0" json:"positions_value"`
	CostBasis       decimal.Decimal `gorm:"column:cost_basis;type:decimal(20,4);not null;default:0" json:"cost_basis"`
	UnrealizedPL    decimal.Decimal `gorm:"column:unrealized_pl;type:decimal(20,4);not null;default:0" json:"unrealized_pl"`
	RealizedPL      decimal.Decimal `gorm:"column:realized_pl;type:decimal(20,4);not null;default:0" json:"realized_pl"`
	TotalReturn     decimal.Decimal `gorm:"column:total_return;type:decimal(10,6);not null;default:0" json:"total_return"`
	DayReturn       decimal.Decimal `gorm:"column:day_return;type:decimal(10,6);not null;default:0" json:"day_return"`
	RiskLevel       string          `gorm:"column:risk_level;type:varchar(10);not null;default:'MODERATE'" json:"risk_level"`
	RebalanceFreq   string          `gorm:"column:rebalance_freq;type:varchar(20);not null;default:'MONTHLY'" json:"rebalance_freq"`
	LastRebalanced  *time.Time      `gorm:"column:last_rebalanced" json:"last_rebalanced"`
}

// PortfolioPosition 组合持仓关联
type PortfolioPosition struct {
	gorm.Model
	PortfolioID     string          `gorm:"column:portfolio_id;type:varchar(32);index;not null" json:"portfolio_id"`
	PositionID      uint            `gorm:"column:position_id;not null" json:"position_id"`
	Symbol          string          `gorm:"column:symbol;type:varchar(20);index;not null" json:"symbol"`
	AssetClass      string          `gorm:"column:asset_class;type:varchar(20);not null" json:"asset_class"`
	Sector          string          `gorm:"column:sector;type:varchar(50)" json:"sector"`
	Weight          decimal.Decimal `gorm:"column:weight;type:decimal(10,6);not null" json:"weight"`
	TargetWeight    decimal.Decimal `gorm:"column:target_weight;type:decimal(10,6);not null" json:"target_weight"`
}

// AssetAllocation 资产配置
type AssetAllocation struct {
	AssetClass   string          `json:"asset_class"`
	Weight       decimal.Decimal `json:"weight"`
	Value        decimal.Decimal `json:"value"`
	TargetWeight decimal.Decimal `json:"target_weight"`
	Deviation    decimal.Decimal `json:"deviation"`
}

// SectorAllocation 行业配置
type SectorAllocation struct {
	Sector       string          `json:"sector"`
	Weight       decimal.Decimal `json:"weight"`
	Value        decimal.Decimal `json:"value"`
	TargetWeight decimal.Decimal `json:"target_weight"`
}

// PortfolioRisk 组合风险指标
type PortfolioRisk struct {
	Volatility       decimal.Decimal `json:"volatility"`
	Beta             decimal.Decimal `json:"beta"`
	SharpeRatio      decimal.Decimal `json:"sharpe_ratio"`
	SortinoRatio     decimal.Decimal `json:"sortino_ratio"`
	MaxDrawdown      decimal.Decimal `json:"max_drawdown"`
	VaR95            decimal.Decimal `json:"var_95"`
	CVaR95           decimal.Decimal `json:"cvar_95"`
	TrackingError    decimal.Decimal `json:"tracking_error"`
	InformationRatio decimal.Decimal `json:"information_ratio"`
}

// RiskAttribution 风险归因
type RiskAttribution struct {
	FactorName     string          `json:"factor_name"`
	FactorExposure decimal.Decimal `json:"factor_exposure"`
	FactorReturn   decimal.Decimal `json:"factor_return"`
	Contribution   decimal.Decimal `json:"contribution"`
}

// BrinsonAttribution Brinson业绩归因
type BrinsonAttribution struct {
	AssetClass        string          `json:"asset_class"`
	AllocationEffect  decimal.Decimal `json:"allocation_effect"`
	SelectionEffect   decimal.Decimal `json:"selection_effect"`
	InteractionEffect decimal.Decimal `json:"interaction_effect"`
	TotalEffect       decimal.Decimal `json:"total_effect"`
}

// PortfolioSnapshot 组合快照
type PortfolioSnapshot struct {
	gorm.Model
	PortfolioID    string          `gorm:"column:portfolio_id;type:varchar(32);index;not null" json:"portfolio_id"`
	UserID         string          `gorm:"column:user_id;type:varchar(32);index;not null" json:"user_id"`
	SnapshotDate   time.Time       `gorm:"column:snapshot_date;type:date;uniqueIndex:idx_portfolio_date;not null" json:"snapshot_date"`
	TotalValue     decimal.Decimal `gorm:"column:total_value;type:decimal(20,4);not null" json:"total_value"`
	CashValue      decimal.Decimal `gorm:"column:cash_value;type:decimal(20,4);not null" json:"cash_value"`
	PositionsValue decimal.Decimal `gorm:"column:positions_value;type:decimal(20,4);not null" json:"positions_value"`
	DayReturn      decimal.Decimal `gorm:"column:day_return;type:decimal(10,6);not null" json:"day_return"`
	TotalReturn    decimal.Decimal `gorm:"column:total_return;type:decimal(10,6);not null" json:"total_return"`
	Currency       string          `gorm:"column:currency;type:varchar(3);not null" json:"currency"`
}

// RebalanceSuggestion 再平衡建议
type RebalanceSuggestion struct {
	Symbol        string          `json:"symbol"`
	AssetClass    string          `json:"asset_class"`
	CurrentWeight decimal.Decimal `json:"current_weight"`
	TargetWeight  decimal.Decimal `json:"target_weight"`
	Deviation     decimal.Decimal `json:"deviation"`
	Action        string          `json:"action"`
	Quantity      decimal.Decimal `json:"quantity"`
	Amount        decimal.Decimal `json:"amount"`
}

func (Portfolio) TableName() string         { return "portfolios" }
func (PortfolioPosition) TableName() string { return "portfolio_positions" }
func (PortfolioSnapshot) TableName() string { return "portfolio_snapshots" }

// NewPortfolio 创建新组合
func NewPortfolio(userID, name, baseCurrency string) *Portfolio {
	return &Portfolio{
		PortfolioID:    generatePortfolioID(),
		UserID:         userID,
		Name:           name,
		BaseCurrency:   baseCurrency,
		TotalValue:     decimal.Zero,
		CashValue:      decimal.Zero,
		PositionsValue: decimal.Zero,
		CostBasis:      decimal.Zero,
		UnrealizedPL:   decimal.Zero,
		RealizedPL:     decimal.Zero,
	}
}

// CreateSnapshot 创建快照
func (p *Portfolio) CreateSnapshot(snapshotDate time.Time) *PortfolioSnapshot {
	return &PortfolioSnapshot{
		PortfolioID:    p.PortfolioID,
		UserID:         p.UserID,
		SnapshotDate:   snapshotDate,
		TotalValue:     p.TotalValue,
		CashValue:      p.CashValue,
		PositionsValue: p.PositionsValue,
		DayReturn:      p.DayReturn,
		TotalReturn:    p.TotalReturn,
		Currency:       p.BaseCurrency,
	}
}

// CalculateRebalance 计算再平衡建议
func CalculateRebalance(positions []PortfolioPosition, totalValue decimal.Decimal, threshold decimal.Decimal) []RebalanceSuggestion {
	suggestions := []RebalanceSuggestion{}
	
	for _, pos := range positions {
		if pos.TargetWeight.IsZero() {
			continue
		}
		
		deviation := pos.Weight.Sub(pos.TargetWeight).Abs()
		if deviation.GreaterThan(threshold) {
			action := "HOLD"
			quantity := decimal.Zero
			
			if pos.Weight.GreaterThan(pos.TargetWeight) {
				action = "SELL"
			} else {
				action = "BUY"
			}
			
			suggestions = append(suggestions, RebalanceSuggestion{
				Symbol:        pos.Symbol,
				AssetClass:    pos.AssetClass,
				CurrentWeight: pos.Weight,
				TargetWeight:  pos.TargetWeight,
				Deviation:     deviation,
				Action:        action,
				Amount:        totalValue.Mul(deviation),
			})
		}
	}
	
	return suggestions
}

// CalculateBrinsonAttribution 计算Brinson业绩归因
func CalculateBrinsonAttribution(portfolioWeights, benchmarkWeights, portfolioReturns, benchmarkReturns map[string]decimal.Decimal) []BrinsonAttribution {
	attributions := []BrinsonAttribution{}
	
	assetClasses := make(map[string]bool)
	for ac := range benchmarkWeights {
		assetClasses[ac] = true
	}
	for ac := range portfolioWeights {
		assetClasses[ac] = true
	}
	
	totalBenchReturn := decimal.Zero
	for ac, w := range benchmarkWeights {
		if r, ok := benchmarkReturns[ac]; ok {
			totalBenchReturn = totalBenchReturn.Add(w.Mul(r))
		}
	}
	
	for assetClass := range assetClasses {
		benchWeight := benchmarkWeights[assetClass]
		benchReturn := benchmarkReturns[assetClass]
		portWeight := portfolioWeights[assetClass]
		portReturn := portfolioReturns[assetClass]
		
		allocationEffect := portWeight.Sub(benchWeight).Mul(benchReturn.Sub(totalBenchReturn))
		selectionEffect := benchWeight.Mul(portReturn.Sub(benchReturn))
		interactionEffect := portWeight.Sub(benchWeight).Mul(portReturn.Sub(benchReturn))
		totalEffect := allocationEffect.Add(selectionEffect).Add(interactionEffect)
		
		attributions = append(attributions, BrinsonAttribution{
			AssetClass:        assetClass,
			AllocationEffect:  allocationEffect,
			SelectionEffect:   selectionEffect,
			InteractionEffect: interactionEffect,
			TotalEffect:       totalEffect,
		})
	}
	
	return attributions
}

// CalculateRiskAttribution 计算风险归因
func CalculateRiskAttribution(weights map[string]decimal.Decimal, factorReturns map[string]decimal.Decimal) []RiskAttribution {
	attributions := []RiskAttribution{}
	
	for symbol, weight := range weights {
		for factorName, factorReturn := range factorReturns {
			contribution := weight.Mul(factorReturn)
			
			attributions = append(attributions, RiskAttribution{
				FactorName:     fmt.Sprintf("%s_%s", symbol, factorName),
				FactorExposure: weight,
				FactorReturn:   factorReturn,
				Contribution:   contribution,
			})
		}
	}
	
	return attributions
}

// generatePortfolioID 生成组合ID
func generatePortfolioID() string {
	return fmt.Sprintf("PF%d", time.Now().UnixNano())
}

// 错误定义
var (
	ErrPortfolioNotFound = errors.New("portfolio not found")
	ErrPositionNotFound  = errors.New("position not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidQuantity   = errors.New("invalid quantity")
)
