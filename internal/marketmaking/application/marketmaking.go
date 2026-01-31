package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
)

// MarketMakingService 做市服务门面，整合命令服务和查询服务
type MarketMakingService struct {
	commandService *MarketMakingCommandService
	queryService   *MarketMakingQueryService
}

// NewMarketMakingService 创建做市服务门面实例
func NewMarketMakingService(
	repo domain.MarketMakingRepository,
	orderSvc domain.OrderClient,
	marketSvc domain.MarketDataClient,
	publisher domain.EventPublisher,
) *MarketMakingService {
	return &MarketMakingService{
		commandService: NewMarketMakingCommandService(repo, orderSvc, marketSvc, publisher),
		queryService:   NewMarketMakingQueryService(repo),
	}
}

// SetStrategy 处理设置做市策略
func (s *MarketMakingService) SetStrategy(ctx context.Context, cmd SetStrategyCommand) (string, error) {
	return s.commandService.SetStrategy(ctx, cmd)
}

// GetStrategy 根据符号获取做市策略
func (s *MarketMakingService) GetStrategy(ctx context.Context, symbol string) (*StrategyDTO, error) {
	return s.queryService.GetStrategy(ctx, symbol)
}

// GetPerformance 根据符号获取做市性能
func (s *MarketMakingService) GetPerformance(ctx context.Context, symbol string) (*PerformanceDTO, error) {
	return s.queryService.GetPerformance(ctx, symbol)
}

// ListStrategies 列出所有做市策略
func (s *MarketMakingService) ListStrategies(ctx context.Context) ([]*StrategyDTO, error) {
	return s.queryService.ListStrategies(ctx)
}

// --- DTO Definitions ---

type SetStrategyCommand struct {
	Symbol       string
	Spread       string
	MinOrderSize string
	MaxOrderSize string
	MaxPosition  string
	Status       string
}

type StrategyDTO struct {
	ID           string
	Symbol       string
	Spread       string
	MinOrderSize string
	MaxOrderSize string
	MaxPosition  string
	Status       string
	CreatedAt    int64
	UpdatedAt    int64
}

type PerformanceDTO struct {
	Symbol      string
	TotalPnL    float64
	TotalVolume float64
	TotalTrades int32
	SharpeRatio float64
	StartTime   int64
	EndTime     int64
}
