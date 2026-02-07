package domain

import "time"

const (
	SimulationCreatedEventType        = "marketsimulation.simulation.created"
	SimulationStartedEventType        = "marketsimulation.simulation.started"
	SimulationStoppedEventType        = "marketsimulation.simulation.stopped"
	SimulationStatusUpdatedEventType  = "marketsimulation.status.updated"
	MarketPriceGeneratedEventType     = "market.price"
	SimulationPriceGeneratedEventType = "marketsimulation.price.generated"
)

// SimulationCreatedEvent 模拟创建事件
type SimulationCreatedEvent struct {
	ScenarioID   string    `json:"scenario_id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Type         string    `json:"type"`
	InitialPrice float64   `json:"initial_price"`
	Volatility   float64   `json:"volatility"`
	Drift        float64   `json:"drift"`
	Timestamp    time.Time `json:"timestamp"`
}

// SimulationStartedEvent 模拟开始事件
type SimulationStartedEvent struct {
	ScenarioID string    `json:"scenario_id"`
	Name       string    `json:"name"`
	Symbol     string    `json:"symbol"`
	Timestamp  time.Time `json:"timestamp"`
}

// SimulationStoppedEvent 模拟停止事件
type SimulationStoppedEvent struct {
	ScenarioID string    `json:"scenario_id"`
	Name       string    `json:"name"`
	Symbol     string    `json:"symbol"`
	Timestamp  time.Time `json:"timestamp"`
}

// MarketSimulationPriceGeneratedEvent 市场模拟价格生成事件
type MarketSimulationPriceGeneratedEvent struct {
	ScenarioID string `json:"scenario_id"`
	Symbol     string `json:"symbol"`
	Price      string `json:"price"`
	Timestamp  int64  `json:"timestamp"`
}

// MarketSimulationStatusUpdatedEvent 市场模拟状态更新事件
type MarketSimulationStatusUpdatedEvent struct {
	ScenarioID string    `json:"scenario_id"`
	Name       string    `json:"name"`
	Symbol     string    `json:"symbol"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}
