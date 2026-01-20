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

// MarketSimulationApplicationService handles simulation lifecycle
type MarketSimulationApplicationService struct {
	repo      domain.SimulationRepository
	publisher domain.MarketDataPublisher
	// In-memory map to keep track of running simulations (cancellation functions)
	runningSims map[string]context.CancelFunc
	mu          sync.Mutex
}

// NewMarketSimulationApplicationService creates a new service instance
func NewMarketSimulationApplicationService(repo domain.SimulationRepository, publisher domain.MarketDataPublisher) *MarketSimulationApplicationService {
	return &MarketSimulationApplicationService{
		repo:        repo,
		publisher:   publisher,
		runningSims: make(map[string]context.CancelFunc),
	}
}

// CreateSimulationConfig creates a new simulation configuration
func (s *MarketSimulationApplicationService) CreateSimulationConfig(ctx context.Context, cmd CreateSimulationCommand) (*SimulationDTO, error) {
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
	return s.toDTO(sim), nil
}

// StartSimulation starts a simulation runner
func (s *MarketSimulationApplicationService) StartSimulation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.runningSims[id]; ok {
		return fmt.Errorf("simulation %s is already running", id)
	}

	sim, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if sim == nil {
		return fmt.Errorf("simulation %s not found", id)
	}

	if err := sim.Start(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, sim); err != nil {
		return err
	}

	// Start background worker
	workerCtx, cancel := context.WithCancel(context.Background())
	s.runningSims[id] = cancel

	go s.runWorker(workerCtx, sim)

	slog.Info("Simulation started", "id", id, "symbol", sim.Symbol)
	return nil
}

// StopSimulation stops a running simulation
func (s *MarketSimulationApplicationService) StopSimulation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, ok := s.runningSims[id]
	if !ok {
		// Even if not running locally, update DB status if needed?
		// But let's check DB first.
	} else {
		cancel()
		delete(s.runningSims, id)
	}

	// Update status in DB
	sim, err := s.repo.Get(ctx, id)
	if err != nil {
		slog.Error("Failed to get simulation to stop", "id", id, "error", err)
		return err
	}
	// ... logic
	sim.Stop()
	if err := s.repo.Save(ctx, sim); err != nil {
		slog.Error("Failed to update simulation status", "id", id, "error", err)
		return err
	}

	slog.Info("Simulation stopped", "id", id)
	return nil
}

// GetSimulation returns a simulation by ID
func (s *MarketSimulationApplicationService) GetSimulation(ctx context.Context, id string) (*SimulationDTO, error) {
	sim, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if sim == nil {
		return nil, nil
	}
	return s.toDTO(sim), nil
}

// ListSimulations returns all simulations
func (s *MarketSimulationApplicationService) ListSimulations(ctx context.Context) ([]*SimulationDTO, error) {
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

// runWorker is the background loop for generating prices
func (s *MarketSimulationApplicationService) runWorker(ctx context.Context, simEntity *domain.Simulation) {
	// TODO: Inject a proper event publisher (kafka)
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

			// Publish to Kafka
			priceDec := decimal.NewFromFloat(currentPrice)
			if err := s.publisher.Publish(ctx, simEntity.Symbol, priceDec); err != nil {
				slog.Error("Failed to publish price", "symbol", simEntity.Symbol, "error", err)
			}

			slog.Info("Tick", "symbol", simEntity.Symbol, "price", currentPrice)
		}
	}
}

func (s *MarketSimulationApplicationService) toDTO(sim *domain.Simulation) *SimulationDTO {
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
