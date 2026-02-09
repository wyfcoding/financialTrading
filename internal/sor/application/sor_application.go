package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialTrading/internal/sor/domain"
)

// CreateSORPlanCommand 创建路由计划命令
type CreateSORPlanCommand struct {
	Symbol   string
	Side     string
	Quantity int64
}

// SORApplicationService SOR 应用服务
type SORApplicationService struct {
	engine domain.SOREngine
	logger *slog.Logger
}

func NewSORApplicationService(engine domain.SOREngine, logger *slog.Logger) *SORApplicationService {
	return &SORApplicationService{
		engine: engine,
		logger: logger,
	}
}

func (s *SORApplicationService) CreateSORPlan(ctx context.Context, cmd CreateSORPlanCommand) (*domain.SORPlan, error) {
	s.logger.Info("generating SOR plan", "symbol", cmd.Symbol, "side", cmd.Side, "quantity", cmd.Quantity)

	// 模拟聚合深度（实际应从基础资产服务或市场行情服务获取）
	depths, err := s.engine.AggregateDepths(ctx, cmd.Symbol)
	if err != nil {
		return nil, err
	}

	plan, err := s.engine.CreateSORPlan(ctx, cmd.Side, cmd.Symbol, cmd.Quantity, depths)
	if err != nil {
		return nil, err
	}

	s.logger.Info("SOR plan generated", "symbol", cmd.Symbol, "routes", len(plan.Routes))
	return plan, nil
}

func (s *SORApplicationService) GetDepths(ctx context.Context, symbol string) ([]*domain.MarketDepth, error) {
	return s.engine.AggregateDepths(ctx, symbol)
}
