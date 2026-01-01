package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// MarketSimulationService 市场模拟门面服务，整合 Manager 和 Query。
type MarketSimulationService struct {
	manager *MarketSimulationManager
	query   *MarketSimulationQuery
}

// NewMarketSimulationService 构造函数。
func NewMarketSimulationService(repo domain.SimulationScenarioRepository, publisher domain.MarketDataPublisher) *MarketSimulationService {
	return &MarketSimulationService{
		manager: NewMarketSimulationManager(repo, publisher),
		query:   NewMarketSimulationQuery(repo),
	}
}

// --- Manager (Writes) ---

func (s *MarketSimulationService) StartSimulation(ctx context.Context, name string, symbol string, simulationType string, parameters string) (string, error) {
	return s.manager.StartSimulation(ctx, name, symbol, simulationType, parameters)
}

func (s *MarketSimulationService) StopSimulation(ctx context.Context, scenarioID string) (bool, error) {
	return s.manager.StopSimulation(ctx, scenarioID)
}

// --- Query (Reads) ---

func (s *MarketSimulationService) GetSimulationStatus(ctx context.Context, scenarioID string) (*domain.SimulationScenario, error) {
	return s.query.GetSimulationStatus(ctx, scenarioID)
}
