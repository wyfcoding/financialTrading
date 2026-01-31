package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// MarketSimulationQueryService 市场模拟查询服务
type MarketSimulationQueryService struct {
	repo domain.SimulationRepository
}

// NewMarketSimulationQueryService 创建市场模拟查询服务实例
func NewMarketSimulationQueryService(
	repo domain.SimulationRepository,
) *MarketSimulationQueryService {
	return &MarketSimulationQueryService{
		repo: repo,
	}
}

// GetSimulation 根据场景ID获取模拟
func (s *MarketSimulationQueryService) GetSimulation(ctx context.Context, scenarioID string) (*SimulationDTO, error) {
	sim, err := s.repo.Get(ctx, scenarioID)
	if err != nil {
		return nil, err
	}
	if sim == nil {
		return nil, nil
	}

	return s.toDTO(sim), nil
}

// ListSimulations 列出所有模拟
func (s *MarketSimulationQueryService) ListSimulations(ctx context.Context) ([]*SimulationDTO, error) {
	sims, err := s.repo.List(ctx, 100)
	if err != nil {
		return nil, err
	}

	dtos := make([]*SimulationDTO, len(sims))
	for i, sim := range sims {
		dtos[i] = s.toDTO(sim)
	}

	return dtos, nil
}

// toDTO 将模拟实体转换为DTO
func (s *MarketSimulationQueryService) toDTO(sim *domain.Simulation) *SimulationDTO {
	return &SimulationDTO{
		ID:           sim.ID,
		ScenarioID:   sim.ScenarioID,
		Name:         sim.Name,
		Symbol:       sim.Symbol,
		Type:         string(sim.Type),
		InitialPrice: sim.InitialPrice,
		Volatility:   sim.Volatility,
		Drift:        sim.Drift,
		IntervalMs:   sim.IntervalMs,
		Status:       string(sim.Status),
		CreatedAt:    sim.CreatedAt,
		Kappa:        sim.Kappa,
		Theta:        sim.Theta,
		VolOfVol:     sim.VolOfVol,
		Rho:          sim.Rho,
		JumpLambda:   sim.JumpLambda,
		JumpMu:       sim.JumpMu,
		JumpSigma:    sim.JumpSigma,
	}
}
