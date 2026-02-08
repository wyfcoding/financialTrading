package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/messagequeue"
)

// MarketSimulationCommandService 市场模拟命令服务
type MarketSimulationCommandService struct {
	repo        domain.SimulationRepository
	readRepo    domain.SimulationReadRepository
	publisher   messagequeue.EventPublisher
	runningSims map[string]context.CancelFunc
	mu          sync.Mutex
}

// NewMarketSimulationCommandService 创建市场模拟命令服务实例
func NewMarketSimulationCommandService(
	repo domain.SimulationRepository,
	readRepo domain.SimulationReadRepository,
	publisher messagequeue.EventPublisher,
) *MarketSimulationCommandService {
	return &MarketSimulationCommandService{
		repo:        repo,
		readRepo:    readRepo,
		publisher:   publisher,
		runningSims: make(map[string]context.CancelFunc),
	}
}

// CreateSimulation 创建模拟
func (s *MarketSimulationCommandService) CreateSimulation(ctx context.Context, cmd CreateSimulationCommand) (*SimulationDTO, error) {
	applySimulationParams(&cmd)
	sim := domain.NewSimulation(cmd.Name, cmd.Symbol, cmd.InitialPrice, cmd.Volatility, cmd.Drift, cmd.IntervalMs)
	if cmd.Type != "" {
		sim.Type = domain.SimulationType(cmd.Type)
	}
	sim.Description = cmd.Description
	sim.Parameters = cmd.Parameters
	sim.Kappa = cmd.Kappa
	sim.Theta = cmd.Theta
	sim.VolOfVol = cmd.VolOfVol
	sim.Rho = cmd.Rho
	sim.JumpLambda = cmd.JumpLambda
	sim.JumpMu = cmd.JumpMu
	sim.JumpSigma = cmd.JumpSigma

	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Save(txCtx, sim); err != nil {
			return err
		}
		if s.publisher != nil {
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
			if err := s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SimulationCreatedEventType, sim.ScenarioID, event); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.readRepo != nil {
		_ = s.readRepo.Save(ctx, sim)
	}
	return toSimulationDTO(sim), nil
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
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Save(txCtx, sim); err != nil {
			return err
		}
		if s.publisher != nil {
			// 发布模拟开始事件
			event := domain.SimulationStartedEvent{
				ScenarioID: sim.ScenarioID,
				Name:       sim.Name,
				Symbol:     sim.Symbol,
				Timestamp:  time.Now(),
			}
			if err := s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SimulationStartedEventType, sim.ScenarioID, event); err != nil {
				return err
			}

			// 发布状态更新事件
			statusEvent := domain.MarketSimulationStatusUpdatedEvent{
				ScenarioID: sim.ScenarioID,
				Name:       sim.Name,
				Symbol:     sim.Symbol,
				Status:     string(sim.Status),
				Timestamp:  time.Now(),
			}
			if err := s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SimulationStatusUpdatedEventType, sim.ScenarioID, statusEvent); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Start background worker
	s.startWorker(sim)

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
	if sim == nil {
		return fmt.Errorf("simulation %s not found", cmd.ScenarioID)
	}

	sim.Stop()
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Save(txCtx, sim); err != nil {
			return err
		}
		if s.publisher != nil {
			// 发布模拟停止事件
			event := domain.SimulationStoppedEvent{
				ScenarioID: sim.ScenarioID,
				Name:       sim.Name,
				Symbol:     sim.Symbol,
				Timestamp:  time.Now(),
			}
			if err := s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SimulationStoppedEventType, sim.ScenarioID, event); err != nil {
				return err
			}

			// 发布状态更新事件
			statusEvent := domain.MarketSimulationStatusUpdatedEvent{
				ScenarioID: sim.ScenarioID,
				Name:       sim.Name,
				Symbol:     sim.Symbol,
				Status:     string(sim.Status),
				Timestamp:  time.Now(),
			}
			if err := s.publisher.PublishInTx(txCtx, contextx.GetTx(txCtx), domain.SimulationStatusUpdatedEventType, sim.ScenarioID, statusEvent); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("Failed to update simulation status", "id", cmd.ScenarioID, "error", err)
		return err
	}
	if s.readRepo != nil {
		_ = s.readRepo.Save(ctx, sim)
	}

	slog.Info("Simulation stopped", "id", cmd.ScenarioID)
	return nil
}

// ResumeRunning 恢复运行中的模拟任务
func (s *MarketSimulationCommandService) ResumeRunning(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	running, err := s.repo.ListRunning(ctx, 1000)
	if err != nil {
		return err
	}
	for _, sim := range running {
		if sim == nil {
			continue
		}
		if _, ok := s.runningSims[sim.ScenarioID]; ok {
			continue
		}
		s.startWorker(sim)
	}
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
				Timestamp:  time.Now().UnixMilli(),
			}
			if s.publisher != nil {
				_ = s.publisher.Publish(context.Background(), domain.MarketPriceGeneratedEventType, simEntity.Symbol, event)
			}

			slog.Info("Tick", "symbol", simEntity.Symbol, "price", currentPrice)
		}
	}
}

func (s *MarketSimulationCommandService) startWorker(simEntity *domain.Simulation) {
	if simEntity == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(context.Background())
	s.runningSims[simEntity.ScenarioID] = cancel
	go s.runWorker(workerCtx, simEntity)
}

type simulationParams struct {
	InitialPrice *float64 `json:"initial_price"`
	Volatility   *float64 `json:"volatility"`
	Drift        *float64 `json:"drift"`
	IntervalMs   *int64   `json:"interval_ms"`
	Kappa        *float64 `json:"kappa"`
	Theta        *float64 `json:"theta"`
	VolOfVol     *float64 `json:"vol_of_vol"`
	Rho          *float64 `json:"rho"`
	JumpLambda   *float64 `json:"jump_lambda"`
	JumpMu       *float64 `json:"jump_mu"`
	JumpSigma    *float64 `json:"jump_sigma"`
}

func applySimulationParams(cmd *CreateSimulationCommand) {
	if cmd == nil {
		return
	}
	// Defaults
	if cmd.InitialPrice == 0 {
		cmd.InitialPrice = 100.0
	}
	if cmd.Volatility == 0 {
		cmd.Volatility = 0.2
	}
	if cmd.Drift == 0 {
		cmd.Drift = 0.05
	}
	if cmd.IntervalMs == 0 {
		cmd.IntervalMs = 1000
	}

	if cmd.Parameters == "" {
		return
	}
	var params simulationParams
	if err := json.Unmarshal([]byte(cmd.Parameters), &params); err != nil {
		return
	}
	if params.InitialPrice != nil {
		cmd.InitialPrice = *params.InitialPrice
	}
	if params.Volatility != nil {
		cmd.Volatility = *params.Volatility
	}
	if params.Drift != nil {
		cmd.Drift = *params.Drift
	}
	if params.IntervalMs != nil {
		cmd.IntervalMs = *params.IntervalMs
	}
	if params.Kappa != nil {
		cmd.Kappa = *params.Kappa
	}
	if params.Theta != nil {
		cmd.Theta = *params.Theta
	}
	if params.VolOfVol != nil {
		cmd.VolOfVol = *params.VolOfVol
	}
	if params.Rho != nil {
		cmd.Rho = *params.Rho
	}
	if params.JumpLambda != nil {
		cmd.JumpLambda = *params.JumpLambda
	}
	if params.JumpMu != nil {
		cmd.JumpMu = *params.JumpMu
	}
	if params.JumpSigma != nil {
		cmd.JumpSigma = *params.JumpSigma
	}
}
