package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// MarketSimulationApplicationService 市场模拟服务门面，整合命令服务和查询服务
type MarketSimulationApplicationService struct {
	commandService *MarketSimulationCommandService
	queryService   *MarketSimulationQueryService
}

// NewMarketSimulationApplicationService 创建市场模拟服务门面实例
func NewMarketSimulationApplicationService(
	repo domain.SimulationRepository,
	publisher domain.EventPublisher,
) *MarketSimulationApplicationService {
	return &MarketSimulationApplicationService{
		commandService: NewMarketSimulationCommandService(repo, publisher),
		queryService:   NewMarketSimulationQueryService(repo),
	}
}

// CreateSimulationConfig 创建模拟配置
func (s *MarketSimulationApplicationService) CreateSimulationConfig(ctx context.Context, cmd CreateSimulationCommand) (*SimulationDTO, error) {
	return s.commandService.CreateSimulation(ctx, cmd)
}

// StartSimulation 开始模拟
func (s *MarketSimulationApplicationService) StartSimulation(ctx context.Context, id string) error {
	cmd := StartSimulationCommand{ScenarioID: id}
	return s.commandService.StartSimulation(ctx, cmd)
}

// StopSimulation 停止模拟
func (s *MarketSimulationApplicationService) StopSimulation(ctx context.Context, id string) error {
	cmd := StopSimulationCommand{ScenarioID: id}
	return s.commandService.StopSimulation(ctx, cmd)
}

// GetSimulation 获取模拟
func (s *MarketSimulationApplicationService) GetSimulation(ctx context.Context, id string) (*SimulationDTO, error) {
	return s.queryService.GetSimulation(ctx, id)
}

// ListSimulations 列出所有模拟
func (s *MarketSimulationApplicationService) ListSimulations(ctx context.Context) ([]*SimulationDTO, error) {
	return s.queryService.ListSimulations(ctx)
}
