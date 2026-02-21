//go:build risk_experimental
// +build risk_experimental

// Package domain 增强风险级联与What-if分析
package domain

import (
	"context"
	"github.com/shopspring/decimal"
)

// CascadeLimit 级联限额模型 (账户 -> 组合 -> 公司)
type CascadeLimit struct {
	AccountLimit   decimal.Decimal
	PortfolioLimit decimal.Decimal
	CompanyLimit   decimal.Decimal
}

// CheckCascade 检查级联限额是否超限
func (l *CascadeLimit) CheckCascade(requestedAmount decimal.Decimal, accCurrent, portCurrent, compCurrent decimal.Decimal) (bool, string) {
	if accCurrent.Add(requestedAmount).GreaterThan(l.AccountLimit) {
		return false, "Account Limit Exceeded"
	}
	if portCurrent.Add(requestedAmount).GreaterThan(l.PortfolioLimit) {
		return false, "Portfolio Limit Exceeded"
	}
	if compCurrent.Add(requestedAmount).GreaterThan(l.CompanyLimit) {
		return false, "Company Limit Exceeded"
	}
	return true, ""
}

// WhatIfAnalysis 假设分析：如果执行此笔交易，对整体 VaR 的影响
func (s *RiskDomainService) WhatIfAnalysis(ctx context.Context, userID string, potentialTrade TradeData) (VaRDelta float64) {
	// 1. 获取当前持仓
	// 2. 模拟加入潜在交易后的新持仓
	// 3. 运行蒙特卡洛模拟
	// 4. 返回 VaR 变动差值
	return 0.05 // 模拟值
}
