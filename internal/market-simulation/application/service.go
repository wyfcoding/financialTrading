// Package application 包含市场模拟服务的用例逻辑
package application

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/wyfcoding/financialTrading/internal/market-simulation/domain"
	"github.com/wyfcoding/pkg/logging"
)

// MarketSimulationService 市场模拟应用服务
// 负责管理模拟场景、生成模拟市场数据并发布
type MarketSimulationService struct {
	repo      domain.SimulationScenarioRepository // 场景仓储接口
	publisher domain.MarketDataPublisher          // 市场数据发布接口
}

// NewMarketSimulationService 创建市场模拟应用服务实例
// repo: 注入的场景仓储实现
// publisher: 注入的市场数据发布实现
func NewMarketSimulationService(repo domain.SimulationScenarioRepository, publisher domain.MarketDataPublisher) *MarketSimulationService {
	return &MarketSimulationService{
		repo:      repo,
		publisher: publisher,
	}
}

// StartSimulation 启动模拟
func (s *MarketSimulationService) StartSimulation(ctx context.Context, name string, symbol string, simulationType string, parameters string) (string, error) {
	// 1. 创建场景
	scenario := &domain.SimulationScenario{
		ID:         uuid.New().String(),
		Name:       name,
		Symbol:     symbol,
		Type:       domain.SimulationType(simulationType),
		Parameters: parameters,
		Status:     domain.SimulationStatusRunning,
		StartTime:  time.Now(),
	}

	if err := s.repo.Save(ctx, scenario); err != nil {
		logging.Error(ctx, "Failed to save simulation scenario",
			"name", name,
			"error", err,
		)
		return "", fmt.Errorf("failed to save scenario: %w", err)
	}

	logging.Info(ctx, "Starting simulation",
		"simulation_id", scenario.ID,
		"symbol", symbol,
		"type", simulationType,
	)

	// 2. 启动模拟协程（简化版）
	go func() {
		// 模拟持续发送数据
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		basePrice := 100.0
		for range ticker.C {
			// 简单的随机漫步
			change := (rand.Float64() - 0.5) * 2
			basePrice += change
			if basePrice < 0 {
				basePrice = 0.1
			}

			// 发送数据
			// 注意：这里使用了 context.Background()，实际应有更好的生命周期管理
			// 实际项目中应该使用带有超时或取消机制的 context
			bgCtx := context.Background()
			if err := s.publisher.Publish(bgCtx, symbol, basePrice); err != nil {
				logging.Error(bgCtx, "Failed to publish market data",
					"symbol", symbol,
					"price", basePrice,
					"error", err,
				)
			}
		}
	}()

	return scenario.ID, nil
}

// StopSimulation 停止模拟
func (s *MarketSimulationService) StopSimulation(ctx context.Context, simulationID string) (bool, error) {
	// 简化实现，仅更新状态
	scenario, err := s.repo.GetByID(ctx, simulationID)
	if err != nil {
		logging.Error(ctx, "Failed to get simulation scenario",
			"simulation_id", simulationID,
			"error", err,
		)
		return false, fmt.Errorf("failed to get scenario: %w", err)
	}
	if scenario == nil {
		return false, fmt.Errorf("scenario not found: %s", simulationID)
	}

	scenario.Status = domain.SimulationStatusStopped
	scenario.EndTime = time.Now()
	if err := s.repo.Save(ctx, scenario); err != nil { // 更新状态
		logging.Error(ctx, "Failed to update simulation status",
			"simulation_id", simulationID,
			"error", err,
		)
		return false, fmt.Errorf("failed to update simulation status: %w", err)
	}

	logging.Info(ctx, "Simulation stopped", "simulation_id", simulationID)
	return true, nil
}

// GetSimulationStatus 获取模拟状态
func (s *MarketSimulationService) GetSimulationStatus(ctx context.Context, simulationID string) (*domain.SimulationScenario, error) {
	scenario, err := s.repo.GetByID(ctx, simulationID)
	if err != nil {
		logging.Error(ctx, "Failed to get simulation scenario",
			"simulation_id", simulationID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get scenario: %w", err)
	}
	return scenario, nil
}
