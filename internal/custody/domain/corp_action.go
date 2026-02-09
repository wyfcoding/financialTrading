// Package domain 提供了公司行为（Corporate Action）处理逻辑。
// 变更说明：实现自动化股息派发（Dividend）与拆股（Stock Split）状态机，支持除权除息日触发及批量持仓更新。
package domain

import (
	"context"
	"time"

	"github.com/wyfcoding/pkg/fsm"
)

// CorpActionType 公司行为类型
type CorpActionType string

const (
	ActionDividend CorpActionType = "DIVIDEND" // 派息
	ActionSplit    CorpActionType = "SPLIT"    // 拆股
	ActionBonus    CorpActionType = "BONUS"    // 送股
)

// CorpAction 实体代表一次公司行为（如某股票的派息计划）
type CorpAction struct {
	ActionID   string                       `json:"action_id"`
	Symbol     string                       `json:"symbol"`
	Type       CorpActionType               `json:"type"`
	Ratio      float64                      `json:"ratio"`       // 拆股比例 (如 1:2 则为 2.0) 或 每股派息金额
	RecordDate time.Time                    `json:"record_date"` // 股权登记日
	ExDate     time.Time                    `json:"ex_date"`     // 除权除息日
	PayDate    time.Time                    `json:"pay_date"`    // 派发日
	Status     string                       `json:"status"`      // ANNOUNCED, EXCUITING, COMPLETED, CANCELLED
	fsm        *fsm.Machine[string, string] `json:"-"`
}

// CorpActionExecution 记录单笔持仓的处理结果
type CorpActionExecution struct {
	ExecutionID string    `json:"execution_id"`
	ActionID    string    `json:"action_id"`
	UserID      uint64    `json:"user_id"`
	OldPosition int64     `json:"old_position"`
	NewPosition int64     `json:"new_position"`
	ChangeAmt   int64     `json:"change_amt"`
	ExecutedAt  time.Time `json:"executed_at"`
}

// CorpActionService 公司行为处理接口
type CorpActionService interface {
	// AnnounceAction 发布公告
	AnnounceAction(ctx context.Context, action *CorpAction) error
	// ExecuteBatch 批量执行（在除权除息日或派发日执行）
	ExecuteBatch(ctx context.Context, actionID string) error
}

func (c *CorpAction) initFSM() {
	m := fsm.NewMachine[string, string](c.Status)
	m.AddTransition("ANNOUNCED", "START", "EXECUTING")
	m.AddTransition("EXECUTING", "FINISH", "COMPLETED")
	m.AddTransition("ANNOUNCED", "CANCEL", "CANCELLED")
	c.fsm = m
}

// ExecuteSplit 执行拆股算法
func (c *CorpAction) ExecuteSplit(oldQty int64) int64 {
	if c.Type != ActionSplit {
		return oldQty
	}
	// 结果向上取整或按比例计算逻辑
	return int64(float64(oldQty) * c.Ratio)
}

// CalculateDividend 计算应派利息
func (c *CorpAction) CalculateDividend(qty int64) int64 {
	if c.Type != ActionDividend {
		return 0
	}
	// Ratio 在此处代表每股派息金（分）
	return int64(float64(qty) * c.Ratio)
}
