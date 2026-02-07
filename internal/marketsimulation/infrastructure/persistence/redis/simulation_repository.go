package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
)

type SimulationRedisRepository struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewSimulationRedisRepository(client redis.UniversalClient) *SimulationRedisRepository {
	return &SimulationRedisRepository{
		client: client,
		prefix: "marketsimulation:scenario:",
		ttl:    10 * time.Minute,
	}
}

func (r *SimulationRedisRepository) Save(ctx context.Context, sim *domain.Simulation) error {
	if sim == nil {
		return nil
	}
	data, err := json.Marshal(sim)
	if err != nil {
		return fmt.Errorf("failed to marshal simulation: %w", err)
	}
	return r.client.Set(ctx, r.key(sim.ScenarioID), data, r.ttl).Err()
}

func (r *SimulationRedisRepository) Get(ctx context.Context, scenarioID string) (*domain.Simulation, error) {
	if scenarioID == "" {
		return nil, nil
	}
	data, err := r.client.Get(ctx, r.key(scenarioID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get simulation from redis: %w", err)
	}
	var sim domain.Simulation
	if err := json.Unmarshal(data, &sim); err != nil {
		return nil, fmt.Errorf("failed to unmarshal simulation: %w", err)
	}
	return &sim, nil
}

func (r *SimulationRedisRepository) key(scenarioID string) string {
	return r.prefix + scenarioID
}
