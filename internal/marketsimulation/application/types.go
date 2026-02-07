package application

import (
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// CreateSimulationCommand defines parameters to start a new simulation config.
type CreateSimulationCommand struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Symbol       string  `json:"symbol"`
	Type         string  `json:"type"` // RANDOM_WALK, HESTON, JUMP_DIFF, REPLAY, SHOCK
	Parameters   string  `json:"parameters"`
	InitialPrice float64 `json:"initial_price"`
	Volatility   float64 `json:"volatility"`
	Drift        float64 `json:"drift"`
	IntervalMs   int64   `json:"interval_ms"`

	// Optional for Heston/Jump
	Kappa      float64 `json:"kappa"`
	Theta      float64 `json:"theta"`
	VolOfVol   float64 `json:"vol_of_vol"`
	Rho        float64 `json:"rho"`
	JumpLambda float64 `json:"jump_lambda"`
	JumpMu     float64 `json:"jump_mu"`
	JumpSigma  float64 `json:"jump_sigma"`
}

type StartSimulationCommand struct {
	ScenarioID string
}

type StopSimulationCommand struct {
	ScenarioID string
}

// SimulationDTO represents the exposed simulation data.
type SimulationDTO struct {
	ID           uint      `json:"id"`
	ScenarioID   string    `json:"scenario_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Symbol       string    `json:"symbol"`
	Type         string    `json:"type"`
	Parameters   string    `json:"parameters"`
	InitialPrice float64   `json:"initial_price"`
	Volatility   float64   `json:"volatility"`
	Drift        float64   `json:"drift"`
	IntervalMs   int64     `json:"interval_ms"`
	Status       string    `json:"status"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Heston/Jump specific
	Kappa      float64 `json:"kappa"`
	Theta      float64 `json:"theta"`
	VolOfVol   float64 `json:"vol_of_vol"`
	Rho        float64 `json:"rho"`
	JumpLambda float64 `json:"jump_lambda"`
	JumpMu     float64 `json:"jump_mu"`
	JumpSigma  float64 `json:"jump_sigma"`
}

func toSimulationDTO(sim *domain.Simulation) *SimulationDTO {
	if sim == nil {
		return nil
	}
	return &SimulationDTO{
		ID:           sim.ID,
		ScenarioID:   sim.ScenarioID,
		Name:         sim.Name,
		Description:  sim.Description,
		Symbol:       sim.Symbol,
		Type:         string(sim.Type),
		Parameters:   sim.Parameters,
		InitialPrice: sim.InitialPrice,
		Volatility:   sim.Volatility,
		Drift:        sim.Drift,
		IntervalMs:   sim.IntervalMs,
		Status:       string(sim.Status),
		StartTime:    sim.StartTime,
		EndTime:      sim.EndTime,
		CreatedAt:    sim.CreatedAt,
		UpdatedAt:    sim.UpdatedAt,
		Kappa:        sim.Kappa,
		Theta:        sim.Theta,
		VolOfVol:     sim.VolOfVol,
		Rho:          sim.Rho,
		JumpLambda:   sim.JumpLambda,
		JumpMu:       sim.JumpMu,
		JumpSigma:    sim.JumpSigma,
	}
}

func toSimulationDTOs(sims []*domain.Simulation) []*SimulationDTO {
	dtos := make([]*SimulationDTO, len(sims))
	for i, sim := range sims {
		dtos[i] = toSimulationDTO(sim)
	}
	return dtos
}
