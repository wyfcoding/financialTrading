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

// MarginRequirement 保证金要求
type MarginRequirement struct {
	AccountID     string
	InitialMargin decimal.Decimal
	MaintMargin   decimal.Decimal
	CurrentMargin decimal.Decimal
	Shortfall     decimal.Decimal
	IsSufficient  bool
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
	CalculateRequirement(ctx context.Context, symbol string, quantity int64, price int64) (*MarginRequirement, error)
}

// DefaultMarginService 默认融资服务实现
type DefaultMarginService struct {
	repo MarginRepository
}

func NewMarginService(repo MarginRepository) MarginService {
	return &DefaultMarginService{repo: repo}
}

func (s *DefaultMarginService) CalculateMarginRatio(ctx context.Context, accountID string) (decimal.Decimal, error) {
	acc, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return decimal.Zero, err
	}
	if acc.BorrowedAmount.IsZero() {
		return decimal.NewFromInt(999), nil
	}
	return acc.CollateralVal.Div(acc.BorrowedAmount), nil
}

func (s *DefaultMarginService) ApplyLoan(ctx context.Context, userID string, amount decimal.Decimal) (*LoanRequest, error) {
	return &LoanRequest{Status: "APPROVED"}, nil
}

func (s *DefaultMarginService) AccrueInterest(ctx context.Context, accountID string) error {
	return nil
}

func (s *DefaultMarginService) CheckLiquidation(ctx context.Context, accountID string) (bool, error) {
	return false, nil
}

func (s *DefaultMarginService) CalculateRequirement(ctx context.Context, symbol string, quantity int64, price int64) (*MarginRequirement, error) {
	val := decimal.NewFromInt(price).Mul(decimal.NewFromInt(quantity))
	initial := val.Mul(decimal.NewFromFloat(0.5)) // 50%
	maint := val.Mul(decimal.NewFromFloat(0.3))   // 30%
	return &MarginRequirement{
		InitialMargin: initial,
		MaintMargin:   maint,
		IsSufficient:  true,
	}, nil
}

// MarginRiskManager 融资风险管理器
type MarginRiskManager struct {
	repo MarginRepository
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

	if acc.BorrowedAmount.IsZero() {
		acc.MarginRatio = decimal.NewFromInt(999)
		acc.Status = MarginStatusNormal
	} else {
		acc.MarginRatio = acc.CollateralVal.Div(acc.BorrowedAmount)
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
