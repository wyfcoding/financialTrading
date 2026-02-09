// 变更说明：新增保证金追缴 (Margin Call) 与违约处理逻辑，确保清算系统的财务稳健。
// 假设：当维持保证金不足时，系统会自动生成保证金追缴通知，并设置 24 小时补缴期限，逾期未补缴则标记为违约。
package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// MarginCallStatus 追缴单状态
type MarginCallStatus string

const (
	MarginCallPending   MarginCallStatus = "PENDING"
	MarginCallFulfilled MarginCallStatus = "FULFILLED"
	MarginCallDefaulted MarginCallStatus = "DEFAULTED"
)

// MarginCall 保证金追缴单
type MarginCall struct {
	CallID         string
	UserID         string
	RequiredAmount decimal.Decimal // 需补缴金额
	Deadline       time.Time
	Status         MarginCallStatus
	CreatedAt      time.Time
}

// DefaultRecord 违约记录
type DefaultRecord struct {
	UserID    string
	Reason    string
	Amount    decimal.Decimal
	Timestamp time.Time
}

// SettlementRiskService 清算风险服务
type SettlementRiskService struct{}

func NewSettlementRiskService() *SettlementRiskService {
	return &SettlementRiskService{}
}

// CreateMarginCall 生成保证金追缴单
func (s *SettlementRiskService) CreateMarginCall(userID string, shortfall decimal.Decimal) *MarginCall {
	return &MarginCall{
		CallID:         fmt.Sprintf("MC-%s-%d", userID, time.Now().Unix()),
		UserID:         userID,
		RequiredAmount: shortfall,
		Deadline:       time.Now().Add(24 * time.Hour),
		Status:         MarginCallPending,
		CreatedAt:      time.Now(),
	}
}

// HandleDefault 处理违约
func (s *SettlementRiskService) HandleDefault(call *MarginCall) *DefaultRecord {
	if time.Now().After(call.Deadline) && call.Status == MarginCallPending {
		call.Status = MarginCallDefaulted
		return &DefaultRecord{
			UserID:    call.UserID,
			Reason:    "Margin call deadline exceeded",
			Amount:    call.RequiredAmount,
			Timestamp: time.Now(),
		}
	}
	return nil
}
