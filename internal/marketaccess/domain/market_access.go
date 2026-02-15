// Package domain 市场准入(Pre-trade Risk)领域模型
package domain

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LimitType 限制类型
type LimitType string

const (
	LimitTypeMaxOrderQty   LimitType = "MAX_ORDER_QTY"
	LimitTypeMaxOrderValue LimitType = "MAX_ORDER_VALUE"
	LimitTypeDailyValue    LimitType = "DAILY_VALUE"
	LimitTypeRestricted    LimitType = "RESTRICTED_SYMBOL" // 禁通过名单
	LimitTypeOnlyReduce    LimitType = "ONLY_REDUCE"       // 仅平仓
)

// TradingPermission 交易权限
type TradingPermission struct {
	gorm.Model
	AccountID   string `gorm:"column:account_id;index;not null"`
	Exchange    string `gorm:"column:exchange;index;not null"`
	InstrumentType string `gorm:"column:instrument_type;not null"` // EQUITY, FUTURE, OPTION
	Enabled     bool   `gorm:"column:enabled;not null;default:true"`
}

func (TradingPermission) TableName() string { return "ma_permissions" }

// TradingLimit 交易限制规则
type TradingLimit struct {
	gorm.Model
	AccountID   string          `gorm:"column:account_id;index"` // 空表示全局
	Symbol      string          `gorm:"column:symbol;index"`     // 空表示全品种
	LimitType   LimitType       `gorm:"column:limit_type;type:varchar(32);not null"`
	LimitValue  decimal.Decimal `gorm:"column:limit_value;type:decimal(20,4)"`
	Currency    string          `gorm:"column:currency;type:char(3)"`
	IsEnabled   bool            `gorm:"column:is_enabled;default:true"`
}

func (TradingLimit) TableName() string { return "ma_limits" }

// CheckRequest 检查请求
type CheckRequest struct {
	AccountID string
	Symbol    string
	Side      string // BUY, SELL
	Quantity  decimal.Decimal
	Price     decimal.Decimal
}

// MarketAccessService 领域服务接口
type MarketAccessService interface {
	CheckAccess(ctx context.Context, req CheckRequest) error
}

// --- Implementation Logic ---

func CheckExample(req CheckRequest, limits []TradingLimit, perm *TradingPermission) error {
	if perm == nil || !perm.Enabled {
		return errors.New("no trading permission")
	}

	value := req.Quantity.Mul(req.Price)

	for _, limit := range limits {
		if !limit.IsEnabled {
			continue
		}
		
		switch limit.LimitType {
		case LimitTypeMaxOrderQty:
			if req.Quantity.GreaterThan(limit.LimitValue) {
				return errors.New("exceeds max order quantity")
			}
		case LimitTypeMaxOrderValue:
			if value.GreaterThan(limit.LimitValue) {
				return errors.New("exceeds max order value")
			}
		case LimitTypeRestricted:
			return errors.New("symbol is restricted")
		}
	}
	return nil
}
