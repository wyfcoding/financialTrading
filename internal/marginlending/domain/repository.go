package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// MarginRepository 保证金仓储接口
type MarginRepository interface {
	GetAccount(ctx context.Context, accountID string) (*MarginAccount, error)
	SaveAccount(ctx context.Context, account *MarginAccount) error
	FindAccountByUserID(ctx context.Context, userID uint64) (*MarginAccount, error)
}

// MarginAccount 保证金账户领域对象 (与核心业务逻辑保持一致)
type MarginAccount struct {
	AccountID       string          `json:"account_id"`
	UserID          uint64          `json:"user_id"`
	CollateralVal   decimal.Decimal `json:"collateral_val"`
	BorrowedAmount  decimal.Decimal `json:"borrowed_amount"`
	InterestAccrued decimal.Decimal `json:"interest_accrued"`
	MarginRatio     decimal.Decimal `json:"margin_ratio"`
	Status          MarginStatus    `json:"status"`
	LeverageLimit   int32           `json:"leverage_limit"`
	LastInterestAt  time.Time       `json:"last_interest_at"`
}
