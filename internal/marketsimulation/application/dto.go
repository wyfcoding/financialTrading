package application

import "time"

// CreateSimulationCommand defines parameters to start a new simulation config
type CreateSimulationCommand struct {
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	Type         string  `json:"type"` // RANDOM_WALK, HESTON, JUMP_DIFF
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

// SimulationDTO represents the exposed simulation data
type SimulationDTO struct {
	ID           uint      `json:"id"`
	ScenarioID   string    `json:"scenario_id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Type         string    `json:"type"`
	InitialPrice float64   `json:"initial_price"`
	Volatility   float64   `json:"volatility"`
	Drift        float64   `json:"drift"`
	IntervalMs   int64     `json:"interval_ms"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`

	// Heston/Jump specific
	Kappa      float64 `json:"kappa"`
	Theta      float64 `json:"theta"`
	VolOfVol   float64 `json:"vol_of_vol"`
	Rho        float64 `json:"rho"`
	JumpLambda float64 `json:"jump_lambda"`
	JumpMu     float64 `json:"jump_mu"`
	JumpSigma  float64 `json:"jump_sigma"`
}
