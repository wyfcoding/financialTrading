package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// MarketSimulationQueryService 市场模拟查询服务
type MarketSimulationQueryService struct {
	repo     domain.SimulationRepository
	readRepo domain.SimulationReadRepository
}

// NewMarketSimulationQueryService 创建市场模拟查询服务实例
func NewMarketSimulationQueryService(
	repo domain.SimulationRepository,
	readRepo domain.SimulationReadRepository,
) *MarketSimulationQueryService {
	return &MarketSimulationQueryService{
		repo:     repo,
		readRepo: readRepo,
	}
}

// GetSimulation 根据场景ID获取模拟
func (s *MarketSimulationQueryService) GetSimulation(ctx context.Context, scenarioID string) (*SimulationDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.Get(ctx, scenarioID); err == nil && cached != nil {
			return toSimulationDTO(cached), nil
		}
	}
	sim, err := s.repo.Get(ctx, scenarioID)
	if err != nil {
		return nil, err
	}
	if sim == nil {
		return nil, nil
	}

	if s.readRepo != nil {
		_ = s.readRepo.Save(ctx, sim)
	}
	return toSimulationDTO(sim), nil
}

// ListSimulations 列出所有模拟
func (s *MarketSimulationQueryService) ListSimulations(ctx context.Context) ([]*SimulationDTO, error) {
	sims, err := s.repo.List(ctx, 100)
	if err != nil {
		return nil, err
	}

	return toSimulationDTOs(sims), nil
}
