// Package domain 提供了融资融券与杠杆交易的领域模型。
// 变更说明：实现融资融券（Margin Lending）核心逻辑，支持抵押品价值评估、利息计提、杠杆倍数控制与强平线监控。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// MarginStatus 融资账户状态
type MarginStatus string

const (
	MarginStatusNormal      MarginStatus = "NORMAL"      // 正常
	MarginStatusWarning     MarginStatus = "WARNING"     // 警戒
	MarginStatusCall        MarginStatus = "CALL"        // 催单
	MarginStatusLiquidating MarginStatus = "LIQUIDATING" // 强平中
)

// MarginAccount 融资账户聚合根
type MarginAccount struct {
	AccountID       string
	UserID          string
	CollateralVal   decimal.Decimal // 抵押品价值
	BorrowedAmount  decimal.Decimal // 已借金额
	InterestAccrued decimal.Decimal // 累计利息
	MarginRatio     decimal.Decimal // 保证金率 (抵押品 / 已借额)
	Status          MarginStatus
	LeverageLimit   int32 // 最大杠杆上限
	LastInterestAt  time.Time
}

// LoanRequest 借款请求
type LoanRequest struct {
	RequestID string
	AccountID string
	Amount    decimal.Decimal
	Asset     string
	Rate      decimal.Decimal
	Status    string // APPROVED, REJECTED
}

// MarginService 融资服务接口
type MarginService interface {
	CalculateMarginRatio(ctx context.Context, accountID string) (decimal.Decimal, error)
	ApplyLoan(ctx context.Context, userID string, amount decimal.Decimal) (*LoanRequest, error)
	AccrueInterest(ctx context.Context, accountID string) error
	CheckLiquidation(ctx context.Context, accountID string) (bool, error)
}

// MarginRiskManager 融资风险管理器
type MarginRiskManager struct {
	repo MarginRepository
}

// MarginRepository 融资仓储接口
type MarginRepository interface {
	GetAccount(ctx context.Context, accountID string) (*MarginAccount, error)
	SaveAccount(ctx context.Context, account *MarginAccount) error
}

func NewMarginRiskManager(repo MarginRepository) *MarginRiskManager {
	return &MarginRiskManager{repo: repo}
}

// EvaluateRisk 评估账户风险并更新状态
func (m *MarginRiskManager) EvaluateRisk(ctx context.Context, accountID string) error {
	acc, err := m.repo.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}

	// 1. 计算当前保证金率
	if acc.BorrowedAmount.IsZero() {
		acc.MarginRatio = decimal.NewFromInt(999) // 无风险
		acc.Status = MarginStatusNormal
	} else {
		acc.MarginRatio = acc.CollateralVal.Div(acc.BorrowedAmount)

		// 2. 根据阈值更新状态
		if acc.MarginRatio.LessThan(decimal.NewFromFloat(1.1)) {
			acc.Status = MarginStatusLiquidating
		} else if acc.MarginRatio.LessThan(decimal.NewFromFloat(1.3)) {
			acc.Status = MarginStatusCall
		} else if acc.MarginRatio.LessThan(decimal.NewFromFloat(1.5)) {
			acc.Status = MarginStatusWarning
		} else {
			acc.Status = MarginStatusNormal
		}
	}

	return m.repo.SaveAccount(ctx, acc)
}

// CalculateAccruedInterest 计算并更新利息
func (acc *MarginAccount) CalculateAccruedInterest(ratePerDay decimal.Decimal) {
	days := decimal.NewFromFloat(time.Since(acc.LastInterestAt).Hours() / 24.0)
	if days.GreaterThan(decimal.Zero) {
		interest := acc.BorrowedAmount.Mul(ratePerDay).Mul(days)
		acc.InterestAccrued = acc.InterestAccrued.Add(interest)
		acc.LastInterestAt = time.Now()
	}
}
