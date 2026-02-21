package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// ExchangeRate 汇率实体
type ExchangeRate struct {
	ID           uint64          `json:"id"`
	FromCurrency string          `json:"from_currency"`
	ToCurrency   string          `json:"to_currency"`
	Pair         string          `json:"pair"` // 格式: USD/CNY
	BaseRate     decimal.Decimal `json:"base_rate"`
	BidRate      decimal.Decimal `json:"bid_rate"`
	AskRate      decimal.Decimal `json:"ask_rate"`
	Source       string          `json:"source"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// RateLock 汇率锁定实体
type RateLock struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	PaymentID    string          `json:"payment_id"`
	FromCurrency string          `json:"from_currency"`
	ToCurrency   string          `json:"to_currency"`
	LockedRate   decimal.Decimal `json:"locked_rate"`
	Amount       decimal.Decimal `json:"amount"`        // 原币种金额
	LockedAmount decimal.Decimal `json:"locked_amount"` // 目标币种金额
	Status       LockStatus      `json:"status"`
	ExpiresAt    time.Time       `json:"expires_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

type LockStatus string

const (
	LockStatusActive  LockStatus = "ACTIVE"
	LockStatusUsed    LockStatus = "USED"
	LockStatusExpired LockStatus = "EXPIRED"
)

// IsValid 检查锁定是否有效
func (l *RateLock) IsValid() bool {
	return l.Status == LockStatusActive && time.Now().Before(l.ExpiresAt)
}

// Use 使用该锁定
func (l *RateLock) Use() {
	l.Status = LockStatusUsed
}

// FiatRepository 仓储接口
type FiatRepository interface {
	SaveRate(ctx context.Context, rate *ExchangeRate) error
	GetRate(ctx context.Context, from, to string) (*ExchangeRate, error)

	SaveLock(ctx context.Context, lock *RateLock) error
	GetLock(ctx context.Context, lockID string) (*RateLock, error)
	UpdateLock(ctx context.Context, lock *RateLock) error
}

// FiatService 领域服务接口 (如果需要复杂的跨实体逻辑)
type FiatDomainService interface {
	CalculateExchange(rate *ExchangeRate, amount decimal.Decimal) decimal.Decimal
}
