package mysql

import (
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"gorm.io/gorm"
)

// SimulationModel 模拟场景数据库模型
type SimulationModel struct {
	gorm.Model
	ScenarioID  string    `gorm:"column:scenario_id;type:varchar(32);uniqueIndex;not null"`
	Name        string    `gorm:"column:name;type:varchar(100);not null"`
	Description string    `gorm:"column:description;type:text"`
	Symbol      string    `gorm:"column:symbol;type:varchar(20);not null"`
	Type        string    `gorm:"column:type;type:varchar(20);not null"`
	Parameters  string    `gorm:"column:parameters;type:text"`
	Status      string    `gorm:"column:status;type:varchar(20);default:'STOPPED'"`
	StartTime   time.Time `gorm:"column:start_time;type:datetime"`
	EndTime     time.Time `gorm:"column:end_time;type:datetime"`

	InitialPrice float64 `gorm:"column:initial_price;type:decimal(20,8)"`
	Volatility   float64 `gorm:"column:volatility;type:decimal(10,4)"`
	Drift        float64 `gorm:"column:drift;type:decimal(10,4)"`
	IntervalMs   int64   `gorm:"column:interval_ms;type:bigint"`

	Kappa      float64 `gorm:"column:kappa;type:decimal(10,4)"`
	Theta      float64 `gorm:"column:theta;type:decimal(10,4)"`
	VolOfVol   float64 `gorm:"column:vol_of_vol;type:decimal(10,4)"`
	Rho        float64 `gorm:"column:rho;type:decimal(10,4)"`
	JumpLambda float64 `gorm:"column:jump_lambda;type:decimal(10,4)"`
	JumpMu     float64 `gorm:"column:jump_mu;type:decimal(10,4)"`
	JumpSigma  float64 `gorm:"column:jump_sigma;type:decimal(10,4)"`
}

func (SimulationModel) TableName() string { return "simulation_scenarios" }

// mapping helpers

func toSimulationModel(sim *domain.Simulation) *SimulationModel {
	if sim == nil {
		return nil
	}
	return &SimulationModel{
		Model: gorm.Model{
			ID:        sim.ID,
			CreatedAt: sim.CreatedAt,
			UpdatedAt: sim.UpdatedAt,
		},
		ScenarioID:   sim.ScenarioID,
		Name:         sim.Name,
		Description:  sim.Description,
		Symbol:       sim.Symbol,
		Type:         string(sim.Type),
		Parameters:   sim.Parameters,
		Status:       string(sim.Status),
		StartTime:    sim.StartTime,
		EndTime:      sim.EndTime,
		InitialPrice: sim.InitialPrice,
		Volatility:   sim.Volatility,
		Drift:        sim.Drift,
		IntervalMs:   sim.IntervalMs,
		Kappa:        sim.Kappa,
		Theta:        sim.Theta,
		VolOfVol:     sim.VolOfVol,
		Rho:          sim.Rho,
		JumpLambda:   sim.JumpLambda,
		JumpMu:       sim.JumpMu,
		JumpSigma:    sim.JumpSigma,
	}
}

func toSimulation(m *SimulationModel) *domain.Simulation {
	if m == nil {
		return nil
	}
	return &domain.Simulation{
		ID:           m.ID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		ScenarioID:   m.ScenarioID,
		Name:         m.Name,
		Description:  m.Description,
		Symbol:       m.Symbol,
		Type:         domain.SimulationType(m.Type),
		Parameters:   m.Parameters,
		Status:       domain.SimulationStatus(m.Status),
		StartTime:    m.StartTime,
		EndTime:      m.EndTime,
		InitialPrice: m.InitialPrice,
		Volatility:   m.Volatility,
		Drift:        m.Drift,
		IntervalMs:   m.IntervalMs,
		Kappa:        m.Kappa,
		Theta:        m.Theta,
		VolOfVol:     m.VolOfVol,
		Rho:          m.Rho,
		JumpLambda:   m.JumpLambda,
		JumpMu:       m.JumpMu,
		JumpSigma:    m.JumpSigma,
	}
}
