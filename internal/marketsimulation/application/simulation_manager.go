package application

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/logging"
)

// MarketSimulationManager 处理所有市场模拟相关的写入操作（Commands）。
type MarketSimulationManager struct {
	repo      domain.SimulationScenarioRepository
	publisher domain.MarketDataPublisher
}

// NewMarketSimulationManager 构造函数。
func NewMarketSimulationManager(repo domain.SimulationScenarioRepository, publisher domain.MarketDataPublisher) *MarketSimulationManager {
	return &MarketSimulationManager{
		repo:      repo,
		publisher: publisher,
	}
}

// StartSimulation 启动模拟
func (m *MarketSimulationManager) StartSimulation(ctx context.Context, name string, symbol string, simulationType string, parameters string) (string, error) {
	scenario := &domain.SimulationScenario{
		ScenarioID: fmt.Sprintf("%d", idgen.GenID()),
		Name:       name,
		Symbol:     symbol,
		Type:       domain.SimulationType(simulationType),
		Parameters: parameters,
		Status:     domain.SimulationStatusRunning,
		StartTime:  time.Now(),
	}

	if err := m.repo.Save(ctx, scenario); err != nil {
		return "", err
	}

	go m.runSimulation(symbol)

	return scenario.ScenarioID, nil
}

// StopSimulation 停止模拟
func (m *MarketSimulationManager) StopSimulation(ctx context.Context, scenarioID string) (bool, error) {
	scenario, err := m.repo.Get(ctx, scenarioID)
	if err != nil || scenario == nil {
		return false, fmt.Errorf("scenario not found")
	}

	scenario.Status = domain.SimulationStatusStopped
	scenario.EndTime = time.Now()
	if err := m.repo.Save(ctx, scenario); err != nil {
		return false, err
	}

	return true, nil
}

func (m *MarketSimulationManager) runSimulation(symbol string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	basePrice := decimal.NewFromFloat(100.0)
	for range ticker.C {
		change := decimal.NewFromFloat((rand.Float64() - 0.5) * 2)
		basePrice = basePrice.Add(change)
		if basePrice.IsNegative() {
			basePrice = decimal.NewFromFloat(0.1)
		}

		if err := m.publisher.Publish(context.Background(), symbol, basePrice); err != nil {
			logging.Error(context.Background(), "SimulationManager: failed to publish price", "symbol", symbol, "error", err)
		}
	}
}
