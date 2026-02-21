//go:build !clearing_experimental
// +build !clearing_experimental

package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/pkg/database"
	"gorm.io/gorm"
)

// 生成摘要：修复并增强结算(Clearing)领域的保证金呼叫(Margin Call)聚合。
// 关键改动：集成 gorm.Model，定义保证金状态机，补全仓储接口。

type MarginCallStatus string

const (
	MarginCallPending    MarginCallStatus = "PENDING"
	MarginCallMet        MarginCallStatus = "MET"
	MarginCallLiquidated MarginCallStatus = "LIQUIDATED"
	MarginCallExpired    MarginCallStatus = "EXPIRED"
)

// MarginCall 保证金呼叫聚合根。
type MarginCall struct {
	gorm.Model
	database.BaseEntity
	AccountID     string           `gorm:"column:account_id;type:varchar(64);index;not null"`
	Amount        decimal.Decimal  `gorm:"column:amount;type:decimal(20,8);not null"`
	RequiredBy    time.Time        `gorm:"column:required_by"`
	Status        MarginCallStatus `gorm:"column:status;type:varchar(20);default:'PENDING'"`
	AutoLiquidate bool             `gorm:"column:auto_liquidate;default:true"`
}

// NettingResult 轧差结果值对象。
type NettingResult struct {
	UserID      string          `json:"user_id"`
	Symbol      string          `json:"symbol"`
	Currency    string          `json:"currency"`
	GrossBuy    decimal.Decimal `json:"gross_buy"`
	GrossSell   decimal.Decimal `json:"gross_sell"`
	NetPosition decimal.Decimal `json:"net_position"`
	NetAmount   decimal.Decimal `json:"net_amount"`
}

// ClearingRepository 结算领域仓储。
type ClearingRepository interface {
	SaveMarginCall(ctx context.Context, mc *MarginCall) error
	FindPendingMarginCalls(ctx context.Context, accountID string) ([]*MarginCall, error)
	// Netting 轧差处理
	PerformNetting(ctx context.Context, batchID string) (*NettingResult, error)
}
