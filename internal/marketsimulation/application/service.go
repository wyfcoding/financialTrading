package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

// MarketSimulationApplicationService handles simulation lifecycle
type MarketSimulationApplicationService struct {
	repo domain.SimulationRepository
	// In-memory map to keep track of running simulations (cancellation functions)
	runningSims map[string]context.CancelFunc
	mu          sync.Mutex
}

// NewMarketSimulationApplicationService creates a new service instance
func NewMarketSimulationApplicationService(repo domain.SimulationRepository) *MarketSimulationApplicationService {
	return &MarketSimulationApplicationService{
		repo:        repo,
		runningSims: make(map[string]context.CancelFunc),
	}
}

// CreateSimulationConfig creates a new simulation configuration
func (s *MarketSimulationApplicationService) CreateSimulationConfig(ctx context.Context, cmd CreateSimulationCommand) (*SimulationDTO, error) {
	sim := domain.NewSimulation(cmd.Name, cmd.Symbol, cmd.InitialPrice, cmd.Volatility, cmd.Drift, cmd.IntervalMs)
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
func (s *MarketSimulationApplicationService) runWorker(ctx context.Context, sim *domain.Simulation) {
	// TODO: Inject a proper event publisher (kafka)
	// For now, we just log the prices
	ticker := time.NewTicker(time.Duration(sim.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	gbm := domain.NewGBM(sim.Drift, sim.Volatility, time.Now().UnixNano())
	currentPrice := sim.InitialPrice

	slog.Info("Worker started", "symbol", sim.Symbol, "initial_price", currentPrice)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopped", "symbol", sim.Symbol)
			return
		case <-ticker.C:
			// Calculate dt in years (IntervalMs / 1000 / 365 / 24 / 60 / 60 approx, or just relative time)
			// Generally Black-Scholes dt is in years.
			dt := float64(sim.IntervalMs) / 1000.0 / (252.0 * 24.0 * 60.0 * 60.0) // simplified trading year seconds
			// Actually let's just use seconds for simplicity if params are per-second, but usually drift/vol are annualized.
			// Assuming user provides Annualized Volatility and Drift.

			currentPrice = gbm.Next(currentPrice, dt)

			// In a real implementation: Publish to Kafka
			// topic: marketdata.quote
			// payload: QuoteDTO
			slog.Info("Tick", "symbol", sim.Symbol, "price", currentPrice)
		}
	}
}

func (s *MarketSimulationApplicationService) toDTO(sim *domain.Simulation) *SimulationDTO {
	return &SimulationDTO{
		ID:           sim.ID,
		Name:         sim.Name,
		Symbol:       sim.Symbol,
		InitialPrice: sim.InitialPrice,
		Volatility:   sim.Volatility,
		Drift:        sim.Drift,
		IntervalMs:   sim.IntervalMs,
		Status:       string(sim.Status),
		CreatedAt:    sim.CreatedAt,
	}
}
