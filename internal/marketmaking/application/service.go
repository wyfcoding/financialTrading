package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingApplicationService 做市服务门面，整合命令服务和查询服务
type MarketMakingApplicationService struct {
	commandService *MarketMakingCommandService
	queryService   *MarketMakingQueryService
}

// NewMarketMakingApplicationService 创建做市服务门面实例
func NewMarketMakingApplicationService(
	repo domain.MarketMakingRepository,
	orderSvc domain.OrderClient,
	marketSvc domain.MarketDataClient,
	publisher domain.EventPublisher,
) *MarketMakingApplicationService {
	return &MarketMakingApplicationService{
		commandService: NewMarketMakingCommandService(repo, orderSvc, marketSvc, publisher),
		queryService:   NewMarketMakingQueryService(repo),
	}
}

// SetStrategy 处理设置做市策略
func (s *MarketMakingApplicationService) SetStrategy(ctx context.Context, cmd SetStrategyCommand) (string, error) {
	return s.commandService.SetStrategy(ctx, cmd)
}

// GetStrategy 根据符号获取做市策略
func (s *MarketMakingApplicationService) GetStrategy(ctx context.Context, symbol string) (*StrategyDTO, error) {
	return s.queryService.GetStrategy(ctx, symbol)
}

// GetPerformance 根据符号获取做市性能
func (s *MarketMakingApplicationService) GetPerformance(ctx context.Context, symbol string) (*PerformanceDTO, error) {
	return s.queryService.GetPerformance(ctx, symbol)
}

// ListStrategies 列出所有做市策略
func (s *MarketMakingApplicationService) ListStrategies(ctx context.Context) ([]*StrategyDTO, error) {
	return s.queryService.ListStrategies(ctx)
}
