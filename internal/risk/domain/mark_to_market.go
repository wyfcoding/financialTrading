// 变更说明：新增实时盯市 (Mark-to-Market) 服务逻辑，支持持仓浮盈浮亏计算与风险预警。
// 假设：行情通过外部 Provider 注入，保证金率低于 5% 时触发强平预警。
package domain

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// --- 盯市服务逻辑 ---

// MTMResult 盯市计算结果
type MTMResult struct {
	UserID        string
	TotalEquity   decimal.Decimal // 总权益 (Available + Frozen + UnPnl)
	UnrealizedPnL decimal.Decimal // 浮动盈亏
	MarginRatio   decimal.Decimal // 保证金率
	Timestamp     time.Time
}

// MarkToMarketService 盯市分析服务
type MarkToMarketService struct {
	mutex sync.RWMutex
}

func NewMarkToMarketService() *MarkToMarketService {
	return &MarkToMarketService{}
}

// CalculatePnL 计算浮动盈亏
func (s *MarkToMarketService) CalculatePnL(assets []PortfolioAsset) decimal.Decimal {
	totalPnL := decimal.Zero
	for _, asset := range assets {
		// PnL = (Current - Average) * Position
		pnl := asset.CurrentPrice.Sub(asset.AveragePrice).Mul(asset.Position)
		totalPnL = totalPnL.Add(pnl)
	}
	return totalPnL
}

// EvaluateRisk 评估保证金风险
func (s *MarkToMarketService) EvaluateRisk(userID string, availableBalance, frozenBalance decimal.Decimal, assets []PortfolioAsset) *MTMResult {
	unPnl := s.CalculatePnL(assets)

	// 总权益 = 可用 + 冻结 + 浮盈亏
	totalEquity := availableBalance.Add(frozenBalance).Add(unPnl)

	// 总持仓名义价值 (Notional Value)
	totalNotional := decimal.Zero
	for _, asset := range assets {
		totalNotional = totalNotional.Add(asset.Position.Abs().Mul(asset.CurrentPrice))
	}

	marginRatio := decimal.NewFromInt(1) // 默认 100%
	if !totalNotional.IsZero() {
		marginRatio = totalEquity.Div(totalNotional)
	}

	return &MTMResult{
		UserID:        userID,
		TotalEquity:   totalEquity,
		UnrealizedPnL: unPnl,
		MarginRatio:   marginRatio,
		Timestamp:     time.Now(),
	}
}

// CheckMaintenanceMargin 检查维持保证金（示例阈值 5%）
func (s *MarkToMarketService) CheckMaintenanceMargin(res *MTMResult) bool {
	threshold := decimal.NewFromFloat(0.05)
	return res.MarginRatio.LessThan(threshold)
}
