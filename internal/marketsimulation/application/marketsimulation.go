package application

import (
	"context"
	"time"

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

// --- DTO Definitions ---

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

type StartSimulationCommand struct {
	ScenarioID string
}

type StopSimulationCommand struct {
	ScenarioID string
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
