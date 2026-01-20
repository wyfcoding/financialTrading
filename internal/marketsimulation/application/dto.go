package application

import "time"

// CreateSimulationCommand defines parameters to start a new simulation config
type CreateSimulationCommand struct {
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	InitialPrice float64 `json:"initial_price"`
	Volatility   float64 `json:"volatility"`
	Drift        float64 `json:"drift"`
	IntervalMs   int64   `json:"interval_ms"`
}

// SimulationDTO represents the exposed simulation data
type SimulationDTO struct {
	ID           uint      `json:"id"`
	ScenarioID   string    `json:"scenario_id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	InitialPrice float64   `json:"initial_price"`
	Volatility   float64   `json:"volatility"`
	Drift        float64   `json:"drift"`
	IntervalMs   int64     `json:"interval_ms"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
