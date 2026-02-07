// Package domain 市场模拟服务的领域模型
package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// SimulationType 模拟类型
type SimulationType string

const (
	SimulationTypeRandomWalk SimulationType = "RANDOM_WALK" // 随机漫步 (GBM)
	SimulationTypeHeston     SimulationType = "HESTON"      // 赫斯顿 (Stochastic Vol)
	SimulationTypeJumpDiff   SimulationType = "JUMP_DIFF"   // 默顿跳跃扩散
	SimulationTypeReplay     SimulationType = "REPLAY"      // 历史回放
	SimulationTypeShock      SimulationType = "SHOCK"       // 市场冲击
)

// SimulationStatus 模拟状态
type SimulationStatus string

const (
	SimulationStatusRunning SimulationStatus = "RUNNING"
	SimulationStatusStopped SimulationStatus = "STOPPED"
)

// Simulation 模拟场景实体
type Simulation struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// ScenarioID 场景唯一标识
	ScenarioID string `json:"scenario_id"`
	// Name 场景名称
	Name string `json:"name"`
	// Description 场景描述
	Description string `json:"description"`
	// Symbol 模拟的交易对
	Symbol string `json:"symbol"`
	// Type 模拟类型
	Type SimulationType `json:"type"`
	// Parameters 模拟参数 (JSON字符串)
	Parameters string `json:"parameters"`
	// Status 模拟状态
	Status SimulationStatus `json:"status"`
	// StartTime 开始时间
	StartTime time.Time `json:"start_time"`
	// EndTime 结束时间
	EndTime time.Time `json:"end_time"`

	// Simulation Parameters (First Class Fields)
	InitialPrice float64 `json:"initial_price"`
	Volatility   float64 `json:"volatility"`
	Drift        float64 `json:"drift"`
	IntervalMs   int64   `json:"interval_ms"`

	// Heston & Jump specific
	Kappa      float64 `json:"kappa"`
	Theta      float64 `json:"theta"`
	VolOfVol   float64 `json:"vol_of_vol"`
	Rho        float64 `json:"rho"`
	JumpLambda float64 `json:"jump_lambda"`
	JumpMu     float64 `json:"jump_mu"`
	JumpSigma  float64 `json:"jump_sigma"`
}

func NewSimulation(name, symbol string, initialPrice, volatility, drift float64, intervalMs int64) *Simulation {
	return &Simulation{
		ScenarioID:   fmt.Sprintf("SIM-%d", time.Now().UnixNano()), // Simple ID generation
		Name:         name,
		Symbol:       symbol,
		Type:         SimulationTypeRandomWalk,
		Status:       SimulationStatusStopped,
		InitialPrice: initialPrice,
		Volatility:   volatility,
		Drift:        drift,
		IntervalMs:   intervalMs,
	}
}

func (s *Simulation) Start() error {
	s.Status = SimulationStatusRunning
	s.StartTime = time.Now()
	return nil
}

func (s *Simulation) Stop() {
	s.Status = SimulationStatusStopped
	s.EndTime = time.Now()
}

// MarketDataPublisher 行情发布接口
type MarketDataPublisher interface {
	Publish(ctx context.Context, symbol string, price decimal.Decimal) error
}

// End of domain file
