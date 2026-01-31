package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// mockEventPublisher 事件发布者的空实现
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishOptionPriced(event domain.OptionPricedEvent) error { return nil }
func (m *mockEventPublisher) PublishGreeksCalculated(event domain.GreeksCalculatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPricingError(event domain.PricingErrorEvent) error { return nil }
func (m *mockEventPublisher) PublishVolatilityUpdated(event domain.VolatilityUpdatedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishPricingModelChanged(event domain.PricingModelChangedEvent) error {
	return nil
}
func (m *mockEventPublisher) PublishBatchPricingCompleted(event domain.BatchPricingCompletedEvent) error {
	return nil
}

// PricingService 定价服务门面，整合命令和查询服务
type PricingService struct {
	Command *PricingCommand
	Query   *PricingQueryService
}

// NewPricingService 构造函数
func NewPricingService(
	repo domain.PricingRepository,
	db interface{},
) (*PricingService, error) {
	// 创建命令服务
	command := NewPricingCommand(
		repo,
	)

	// 创建查询服务
	query := NewPricingQueryService(repo)

	return &PricingService{
		Command: command,
		Query:   query,
	}, nil
}

// --- Command (Writes) ---

// PriceOption 期权定价
func (s *PricingService) PriceOption(ctx context.Context, cmd PriceOptionCommand) (*domain.PricingResult, error) {
	return s.Command.PriceOption(ctx, cmd)
}

// UpdateVolatility 更新波动率
func (s *PricingService) UpdateVolatility(ctx context.Context, cmd UpdateVolatilityCommand) error {
	return s.Command.UpdateVolatility(ctx, cmd)
}

// ChangePricingModel 变更定价模型
func (s *PricingService) ChangePricingModel(ctx context.Context, cmd ChangePricingModelCommand) error {
	return s.Command.ChangePricingModel(ctx, cmd)
}

// BatchPriceOptions 批量定价
func (s *PricingService) BatchPriceOptions(ctx context.Context, cmd BatchPriceOptionsCommand) (*BatchPricingResult, error) {
	return s.Command.BatchPriceOptions(ctx, cmd)
}

// --- Query (Reads) ---

// GetGreeks 计算希腊字母
func (s *PricingService) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice interface{}, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	// 这里需要根据实际类型转换 underlyingPrice
	// 暂时留空，实际应用中需要实现类型转换
	return nil, nil
}

// GetLatestResult 获取最新定价结果
func (s *PricingService) GetLatestResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	return s.Query.GetLatestResult(ctx, symbol)
}

// GetPrice 获取最新价格
func (s *PricingService) GetPrice(ctx context.Context, symbol string) (*PriceDTO, error) {
	return s.Query.GetPrice(ctx, symbol)
}

// GetOptionPrice 获取期权价格
func (s *PricingService) GetOptionPrice(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility float64, riskFreeRate float64) (decimal.Decimal, error) {
	// 转换参数为 PriceOptionCommand
	cmd := PriceOptionCommand{
		Symbol:          contract.Symbol,
		OptionType:      string(contract.Type),
		StrikePrice:     contract.StrikePrice.InexactFloat64(),
		ExpiryDate:      contract.ExpiryDate,
		UnderlyingPrice: underlyingPrice.InexactFloat64(),
		Volatility:      volatility,
		RiskFreeRate:    riskFreeRate,
		DividendYield:   0,
		PricingModel:    "BlackScholes",
	}

	// 调用现有的 PriceOption 方法
	result, err := s.PriceOption(ctx, cmd)
	if err != nil {
		return decimal.Zero, err
	}

	return result.OptionPrice, nil
}

// ListPrices 列出多个符号的最新价格
func (s *PricingService) ListPrices(ctx context.Context, symbols []string) ([]*PriceDTO, error) {
	return s.Query.ListPrices(ctx, symbols)
}

// --- DTO Definitions ---

type PriceDTO struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Mid       float64   `json:"mid"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}
