// Package domain 市场模拟服务的领域模型
package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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
	gorm.Model
	// ScenarioID 场景唯一标识
	ScenarioID string `gorm:"column:scenario_id;type:varchar(32);uniqueIndex;not null" json:"scenario_id"`
	// Name 场景名称
	Name string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// Description 场景描述
	Description string `gorm:"column:description;type:text" json:"description"`
	// Symbol 模拟的交易对
	Symbol string `gorm:"column:symbol;type:varchar(20);not null" json:"symbol"`
	// Type 模拟类型
	Type SimulationType `gorm:"column:type;type:varchar(20);not null" json:"type"`
	// Parameters 模拟参数 (JSON字符串)
	Parameters string `gorm:"column:parameters;type:text" json:"parameters"`
	// Status 模拟状态
	Status SimulationStatus `gorm:"column:status;type:varchar(20);default:'STOPPED'" json:"status"`
	// StartTime 开始时间
	StartTime time.Time `gorm:"column:start_time;type:datetime" json:"start_time"`
	// EndTime 结束时间
	EndTime time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`

	// Simulation Parameters (First Class Fields)
	InitialPrice float64 `gorm:"column:initial_price;type:decimal(20,8)" json:"initial_price"`
	Volatility   float64 `gorm:"column:volatility;type:decimal(10,4)" json:"volatility"`
	Drift        float64 `gorm:"column:drift;type:decimal(10,4)" json:"drift"`
	IntervalMs   int64   `gorm:"column:interval_ms;type:bigint" json:"interval_ms"`

	// Heston & Jump specific
	Kappa      float64 `gorm:"column:kappa;type:decimal(10,4)" json:"kappa"`
	Theta      float64 `gorm:"column:theta;type:decimal(10,4)" json:"theta"`
	VolOfVol   float64 `gorm:"column:vol_of_vol;type:decimal(10,4)" json:"vol_of_vol"`
	Rho        float64 `gorm:"column:rho;type:decimal(10,4)" json:"rho"`
	JumpLambda float64 `gorm:"column:jump_lambda;type:decimal(10,4)" json:"jump_lambda"`
	JumpMu     float64 `gorm:"column:jump_mu;type:decimal(10,4)" json:"jump_mu"`
	JumpSigma  float64 `gorm:"column:jump_sigma;type:decimal(10,4)" json:"jump_sigma"`
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

// SimulationRepository 模拟仓储接口
type SimulationRepository interface {
	Save(ctx context.Context, s *Simulation) error
	Get(ctx context.Context, scenarioID string) (*Simulation, error)
	List(ctx context.Context, limit int) ([]*Simulation, error)
}

// MarketDataPublisher 行情发布接口
type MarketDataPublisher interface {
	Publish(ctx context.Context, symbol string, price decimal.Decimal) error
}

// End of domain file
