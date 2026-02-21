//go:build legacy_corporate_action

// 变更说明：
// 托管服务 (Custody) 收编 CorporateAction (公司行动，如：股票拆股、分红、发息)。
package domain

import (
	"context"
	"errors"
	"github.com/shopspring/decimal"
	"time"
)

var (
	ErrActionAlreadyExecuted = errors.New("corporate action already executed")
)

type ActionType string

const (
	ActionDividend ActionType = "DIVIDEND" // 分红
	ActionSplit    ActionType = "SPLIT"    // 拆股
	ActionMerger   ActionType = "MERGER"   // 合并
)

// CorporateAction 公司行动抽象。影响托管资金库中的标的物数量或产生派息。
type CorporateAction struct {
	ID       uint64
	Symbol   string
	Type     ActionType
	Ratio    decimal.Decimal // 如，10送2 Ratio=0.2, 拆股 1拆2 Ratio=2.0
	CashAmt  decimal.Decimal // 派发现金/股
	ExDate   time.Time       // 除权除息日
	Executed bool
}

// UserPosition 在 Custody 服务挂载的用户托管底仓
type UserPosition struct {
	UserID   uint64
	Symbol   string
	Quantity decimal.Decimal
}

// ApplyCorporateAction 会将派发的持仓改变或派发股息直接反馈回去
func (ca *CorporateAction) Apply(position *UserPosition) (cashPayout decimal.Decimal, newQty decimal.Decimal, err error) {
	if ca.Executed {
		return decimal.Zero, position.Quantity, ErrActionAlreadyExecuted
	}

	switch ca.Type {
	case ActionDividend:
		// 分红: 持仓不变，发放现金 = 数量 * 每股分红
		cashPayout = position.Quantity.Mul(ca.CashAmt)
		newQty = position.Quantity
	case ActionSplit:
		// 拆股: 如 1拆2，数量翻倍
		cashPayout = decimal.Zero
		newQty = position.Quantity.Mul(ca.Ratio)
	default:
		newQty = position.Quantity
	}
	return cashPayout, newQty, nil
}

type CustodyRepository interface {
	SavePosition(ctx context.Context, p *UserPosition) error
	ListPositionsBySymbol(ctx context.Context, symbol string) ([]*UserPosition, error)
}
