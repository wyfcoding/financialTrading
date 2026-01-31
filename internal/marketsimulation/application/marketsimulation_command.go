package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// StartSimulationCommand 开始模拟命令
type StartSimulationCommand struct {
	ScenarioID string
}

// StopSimulationCommand 停止模拟命令
type StopSimulationCommand struct {
	ScenarioID string
}

// MarketSimulationCommandService 市场模拟命令服务
type MarketSimulationCommandService struct {
	repo       domain.SimulationRepository
	publisher  domain.EventPublisher
	runningSims map[string]context.CancelFunc
	mu         sync.Mutex
}

// NewMarketSimulationCommandService 创建市场模拟命令服务实例
func NewMarketSimulationCommandService(
	repo domain.SimulationRepository,
	publisher domain.EventPublisher,
) *MarketSimulationCommandService {
	return &MarketSimulationCommandService{
		repo:       repo,
		publisher:  publisher,
		runningSims: make(map[string]context.CancelFunc),
	}
}

// CreateSimulation 创建模拟
func (s *MarketSimulationCommandService) CreateSimulation(ctx context.Context, cmd CreateSimulationCommand) (*SimulationDTO, error) {
	sim := domain.NewSimulation(cmd.Name, cmd.Symbol, cmd.InitialPrice, cmd.Volatility, cmd.Drift, cmd.IntervalMs)
	sim.Type = domain.SimulationType(cmd.Type)
	sim.Kappa = cmd.Kappa
	sim.Theta = cmd.Theta
	sim.VolOfVol = cmd.VolOfVol
	sim.Rho = cmd.Rho
	sim.JumpLambda = cmd.JumpLambda
	sim.JumpMu = cmd.JumpMu
	sim.JumpSigma = cmd.JumpSigma

	if err := s.repo.Save(ctx, sim); err != nil {
		return nil, err
	}

	// 发布模拟创建事件
	event := domain.SimulationCreatedEvent{
		ScenarioID:   sim.ScenarioID,
		Name:         sim.Name,
		Symbol:       sim.Symbol,
		Type:         string(sim.Type),
		InitialPrice: sim.InitialPrice,
		Volatility:   sim.Volatility,
		Drift:        sim.Drift,
		Timestamp:    time.Now(),
	}
	s.publisher.Publish(ctx, "marketsimulation.simulation.created", sim.ScenarioID, event)

	return s.toDTO(sim), nil
}

// StartSimulation 开始模拟
func (s *MarketSimulationCommandService) StartSimulation(ctx context.Context, cmd StartSimulationCommand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.runningSims[cmd.ScenarioID]; ok {
		return fmt.Errorf("simulation %s is already running", cmd.ScenarioID)
	}

	sim, err := s.repo.Get(ctx, cmd.ScenarioID)
	if err != nil {
		return err
	}
	if sim == nil {
		return fmt.Errorf("simulation %s not found", cmd.ScenarioID)
	}

	if err := sim.Start(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, sim); err != nil {
		return err
	}

	// 发布模拟开始事件
	event := domain.SimulationStartedEvent{
		ScenarioID: sim.ScenarioID,
		Name:       sim.Name,
		Symbol:     sim.Symbol,
		Timestamp:  time.Now(),
	}
	s.publisher.Publish(ctx, "marketsimulation.simulation.started", sim.ScenarioID, event)

	// 发布状态更新事件
	statusEvent := domain.MarketSimulationStatusUpdatedEvent{
		ScenarioID: sim.ScenarioID,
		Name:       sim.Name,
		Symbol:     sim.Symbol,
		Status:     string(sim.Status),
		Timestamp:  time.Now(),
	}
	s.publisher.Publish(ctx, "marketsimulation.status.updated", sim.ScenarioID, statusEvent)

	// Start background worker
	workerCtx, cancel := context.WithCancel(context.Background())
	s.runningSims[cmd.ScenarioID] = cancel

	go s.runWorker(workerCtx, sim)

	slog.Info("Simulation started", "id", cmd.ScenarioID, "symbol", sim.Symbol)
	return nil
}

// StopSimulation 停止模拟
func (s *MarketSimulationCommandService) StopSimulation(ctx context.Context, cmd StopSimulationCommand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, ok := s.runningSims[cmd.ScenarioID]
	if ok {
		cancel()
		delete(s.runningSims, cmd.ScenarioID)
	}

	// Update status in DB
	sim, err := s.repo.Get(ctx, cmd.ScenarioID)
	if err != nil {
		slog.Error("Failed to get simulation to stop", "id", cmd.ScenarioID, "error", err)
		return err
	}

	sim.Stop()
	if err := s.repo.Save(ctx, sim); err != nil {
		slog.Error("Failed to update simulation status", "id", cmd.ScenarioID, "error", err)
		return err
	}

	// 发布模拟停止事件
	event := domain.SimulationStoppedEvent{
		ScenarioID: sim.ScenarioID,
		Name:       sim.Name,
		Symbol:     sim.Symbol,
		Timestamp:  time.Now(),
	}
	s.publisher.Publish(ctx, "marketsimulation.simulation.stopped", sim.ScenarioID, event)

	// 发布状态更新事件
	statusEvent := domain.MarketSimulationStatusUpdatedEvent{
		ScenarioID: sim.ScenarioID,
		Name:       sim.Name,
		Symbol:     sim.Symbol,
		Status:     string(sim.Status),
		Timestamp:  time.Now(),
	}
	s.publisher.Publish(ctx, "marketsimulation.status.updated", sim.ScenarioID, statusEvent)

	slog.Info("Simulation stopped", "id", cmd.ScenarioID)
	return nil
}

// runWorker 运行模拟工作器
func (s *MarketSimulationCommandService) runWorker(ctx context.Context, simEntity *domain.Simulation) {
	ticker := time.NewTicker(time.Duration(simEntity.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	var gen domain.PriceGenerator
	switch simEntity.Type {
	case domain.SimulationTypeHeston:
		gen = domain.NewHestonGenerator(simEntity.InitialPrice, simEntity.Volatility, simEntity.Kappa, simEntity.Theta, simEntity.VolOfVol, simEntity.Rho)
	case domain.SimulationTypeJumpDiff:
		gen = domain.NewJumpDiffusionGenerator(simEntity.InitialPrice, simEntity.Drift, simEntity.Volatility, simEntity.JumpLambda, simEntity.JumpMu, simEntity.JumpSigma)
	default:
		gen = domain.NewGBM(simEntity.Drift, simEntity.Volatility, time.Now().UnixNano())
	}

	currentPrice := simEntity.InitialPrice
	slog.Info("Worker started", "symbol", simEntity.Symbol, "type", simEntity.Type, "initial_price", currentPrice)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "symbol", simEntity.Symbol)
			return
		case <-ticker.C:
			// dt in years. 252 trading days, 24h/day (assuming continuous for crypto or standard for equity).
			// Adjusting to a more standard 1 year = 252 * 24 * 3600 seconds.
			dt := float64(simEntity.IntervalMs) / 1000.0 / (252.0 * 24.0 * 3600.0)

			currentPrice = gen.Next(currentPrice, dt)

			// 发布价格生成事件
			priceDec := decimal.NewFromFloat(currentPrice)
			event := domain.MarketSimulationPriceGeneratedEvent{
				ScenarioID: simEntity.ScenarioID,
				Symbol:     simEntity.Symbol,
				Price:      priceDec.String(),
				Timestamp:  time.Now(),
			}
			s.publisher.Publish(context.Background(), "marketsimulation.price.generated", simEntity.Symbol, event)

			slog.Info("Tick", "symbol", simEntity.Symbol, "price", currentPrice)
		}
	}
}

// toDTO 将模拟实体转换为DTO
func (s *MarketSimulationCommandService) toDTO(sim *domain.Simulation) *SimulationDTO {
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
