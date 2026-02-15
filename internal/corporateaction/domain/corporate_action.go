// Package domain 公司行动领域模型
// 生成摘要：
// 1) 定义 CorporateAction 聚合根，支持多种行动类型（分红、拆股等）
// 2) 定义 Entitlement（权益计算结果）和 Election（用户选择）
// 3) 处理复杂的生命周期：公告日 -> 除权日 -> 股权登记日 -> 支付日
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ActionType string

const (
	ActionTypeCashDividend   ActionType = "CASH_DIVIDEND"   // 现金分红
	ActionTypeStockDividend  ActionType = "STOCK_DIVIDEND"  // 股票分红
	ActionTypeStockSplit     ActionType = "STOCK_SPLIT"     // 拆股
	ActionTypeReverseSplit   ActionType = "REVERSE_SPLIT"   // 合股
	ActionTypeRightsIssue    ActionType = "RIGHTS_ISSUE"    // 配股
	ActionTypeSpinOff        ActionType = "SPIN_OFF"        // 分拆上市
	ActionTypeMerger         ActionType = "MERGER"          // 合并
	ActionTypeTenderOffer    ActionType = "TENDER_OFFER"    // 要约收购
)

type ActionStatus string

const (
	ActionStatusAnnounced ActionStatus = "ANNOUNCED" // 已公告
	ActionStatusActive    ActionStatus = "ACTIVE"    // 进行中
	ActionStatusProcessed ActionStatus = "PROCESSED" // 权益已计算
	ActionStatusCompleted ActionStatus = "COMPLETED" // 支付已完成
	ActionStatusCancelled ActionStatus = "CANCELLED" // 已取消
)

// CorporateAction 公司行动主记录
type CorporateAction struct {
	gorm.Model
	EventID          string          `gorm:"column:event_id;type:varchar(64);uniqueIndex;not null"`
	Symbol           string          `gorm:"column:symbol;type:varchar(32);not null"` // 标的证券
	Type             ActionType      `gorm:"column:type;type:varchar(32);not null"`
	Status           ActionStatus    `gorm:"column:status;type:varchar(32);not null;default:'ANNOUNCED'"`
	
	// 关键日期
	AnnouncementDate time.Time       `gorm:"column:announcement_date"` // 公告日
	ExDate           time.Time       `gorm:"column:ex_date"`           // 除权日
	RecordDate       time.Time       `gorm:"column:record_date"`       // 登记日
	PaymentDate      time.Time       `gorm:"column:payment_date"`      // 支付日
	ElectionDeadline *time.Time      `gorm:"column:election_deadline"` // 选举截止日（可选事件）
	
	// 比例参数
	RatioNumerator   decimal.Decimal `gorm:"column:ratio_numerator;type:decimal(20,8)"`   // 分子
	RatioDenominator decimal.Decimal `gorm:"column:ratio_denominator;type:decimal(20,8)"` // 分母
	Price            decimal.Decimal `gorm:"column:price;type:decimal(20,4)"`             // 配股价/收购价
	Currency         string          `gorm:"column:currency;type:char(3)"`
	
	Description      string          `gorm:"column:description;type:text"`
	
	// 关联
	Entitlements     []Entitlement   `gorm:"foreignKey:ActionID" json:"entitlements,omitempty"`
}

func (CorporateAction) TableName() string { return "ca_actions" }

// Entitlement 权益记录（每个持有人的权益计算结果）
type Entitlement struct {
	gorm.Model
	EntitlementID string          `gorm:"column:entitlement_id;type:varchar(64);uniqueIndex;not null"`
	ActionID      uint            `gorm:"column:action_id;index;not null"`
	AccountID     string          `gorm:"column:account_id;index;not null"` // 持有人账户
	HoldingQty    decimal.Decimal `gorm:"column:holding_qty;type:decimal(20,4);not null"` // 登记日持仓
	
	// 权益结果
	PayoutCash    decimal.Decimal `gorm:"column:payout_cash;type:decimal(20,4)"`    // 应付现金
	PayoutStock   decimal.Decimal `gorm:"column:payout_stock;type:decimal(20,4)"`   // 应付股票
	StockSymbol   string          `gorm:"column:stock_symbol;type:varchar(32)"`     // 股票代码（分拆时可能不同）
	
	Status        string          `gorm:"column:status;type:varchar(32);default:'CALCULATED'"` // CALCULATED, PAID, FAILED
	
	// 选举（针对可选事件）
	ElectionOption string         `gorm:"column:election_option;type:varchar(32)"`
}

func (Entitlement) TableName() string { return "ca_entitlements" }

// NewCorporateAction 创建新的公司行动
func NewCorporateAction(eventID, symbol string, typ ActionType) *CorporateAction {
	return &CorporateAction{
		EventID: eventID,
		Symbol:  symbol,
		Type:    typ,
		Status:  ActionStatusAnnounced,
	}
}

// CalculateEntitlement 计算单个账户的权益
func (a *CorporateAction) CalculateEntitlement(accountID string, holdingQty decimal.Decimal) (*Entitlement, error) {
	e := &Entitlement{
		ActionID:   a.ID,
		AccountID:  accountID,
		HoldingQty: holdingQty,
		Status:     "CALCULATED",
	}

	switch a.Type {
	case ActionTypeCashDividend:
		// 每股派息：持仓 * (分子/分母)
		rate := a.RatioNumerator.Div(a.RatioDenominator)
		e.PayoutCash = holdingQty.Mul(rate)
		
	case ActionTypeStockDividend, ActionTypeStockSplit:
		// 送股/拆股：持仓 * (分子/分母)
		rate := a.RatioNumerator.Div(a.RatioDenominator)
		e.PayoutStock = holdingQty.Mul(rate)
		e.StockSymbol = a.Symbol
		
	case ActionTypeReverseSplit:
		// 合股：持仓 * (分子/分母) -> 通常分子<分母
		rate := a.RatioNumerator.Div(a.RatioDenominator)
		e.PayoutStock = holdingQty.Mul(rate)
		e.StockSymbol = a.Symbol
		// 处理碎股 cash-in-lieu (简化略过)
		
	default:
		return nil, errors.New("unsupported action type for auto calculation")
	}

	return e, nil
}

// Repository 接口
type ActionRepository interface {
	Save(ctx context.Context, action *CorporateAction) error
	GetByEventID(ctx context.Context, eventID string) (*CorporateAction, error)
	ListActive(ctx context.Context, date time.Time) ([]*CorporateAction, error)
}

type EntitlementRepository interface {
	Save(ctx context.Context, ent *Entitlement) error
	ListByActionID(ctx context.Context, actionID uint) ([]*Entitlement, error)
	GetByAccountAndAction(ctx context.Context, accountID string, actionID uint) (*Entitlement, error)
}
