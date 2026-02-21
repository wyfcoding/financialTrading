// 生成摘要：从 simulation 合并到 marketsimulation 域。
// 整合 marketreplay 历史逐笔回放与量化训练模拟。
package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// SimState 模拟测试状态。
type SimState string

const (
	SimStateRunning SimState = "RUNNING"
	SimStatePaused  SimState = "PAUSED"
	SimStateDone    SimState = "DONE"
	SimStateError   SimState = "ERROR"
)

// SimulationSession 沙箱模拟交易/历史回放会话。
// 拥有独立影子账户与影子订单簿。
type SimulationSession struct {
	// SessionID 会话 ID。
	SessionID string `json:"session_id"`
	// AlgorithmID 加载的量化策略 ID。
	AlgorithmID string `json:"algorithm_id"`
	// StartTime 回溯历史起点。
	StartTime time.Time `json:"start_time"`
	// EndTime 回溯历史终点。
	EndTime time.Time `json:"end_time"`
	// CurrentTime 虚拟时钟刻度。
	CurrentTime time.Time `json:"current_time"`
	// Speed 播放倍速（如 10x）。
	Speed float64 `json:"speed"`
	// InitialCap 初始影子资金。
	InitialCap decimal.Decimal `json:"initial_cap"`
	// CurrentCap 当前资金净值（NAV）。
	CurrentCap decimal.Decimal `json:"current_cap"`
	// Status 会话状态。
	Status SimState `json:"status"`
}

// SimTick 行情 Tick 数据。
type SimTick struct {
	Price  decimal.Decimal `json:"price"`
	Volume decimal.Decimal `json:"volume"`
	Time   time.Time       `json:"time"`
}

// SimOrderBookSnapshot 订单簿快照。
type SimOrderBookSnapshot struct {
	Bids [50]struct{ Price, Qty decimal.Decimal }
	Asks [50]struct{ Price, Qty decimal.Decimal }
}

// Step 时钟推演：推到下个事件节点并计算未实现盈亏。
func (s *SimulationSession) Step(nextTime time.Time, pnlDelta decimal.Decimal) {
	if s.Status == SimStateRunning {
		s.CurrentTime = nextTime
		s.CurrentCap = s.CurrentCap.Add(pnlDelta)
		if s.CurrentTime.After(s.EndTime) {
			s.Status = SimStateDone
		}
	}
}

// TickReplayer 数据回放接口，从 ClickHouse 抽数据喂给策略模型。
type TickReplayer interface {
	FetchTicks(ctx context.Context, symbol string, from, to time.Time) ([]SimTick, error)
	FetchOrderBookSnapshot(ctx context.Context, symbol string, at time.Time) (*SimOrderBookSnapshot, error)
}
