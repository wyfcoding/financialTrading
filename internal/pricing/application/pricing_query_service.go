package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

// PricingQueryService 处理所有定价相关的查询操作（Queries）。
type PricingQueryService struct {
	repo domain.PricingRepository
}

// NewPricingQueryService 构造函数。
func NewPricingQueryService(repo domain.PricingRepository) *PricingQueryService {
	return &PricingQueryService{
		repo: repo,
	}
}

// GetGreeks 计算希腊字母
func (s *PricingQueryService) GetGreeks(ctx context.Context, contract domain.OptionContract, underlyingPrice decimal.Decimal, volatility, riskFreeRate float64) (*domain.Greeks, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	timeToExpiry := float64(contract.ExpiryDate-time.Now().UnixMilli()) / 1000 / 24 / 3600 / 365
	if timeToExpiry < 0 {
		return &domain.Greeks{
			Delta: decimal.Zero,
			Gamma: decimal.Zero,
			Theta: decimal.Zero,
			Vega:  decimal.Zero,
			Rho:   decimal.Zero,
		}, nil
	}

	sVal, _ := underlyingPrice.Float64()
	kVal, _ := contract.StrikePrice.Float64()

	result := domain.CalculateBlackScholes(contract.Type, domain.BlackScholesInput{
		S: sVal,
		K: kVal,
		T: timeToExpiry,
		R: riskFreeRate,
		V: volatility,
	})

	return &domain.Greeks{
		Delta: result.Delta,
		Gamma: result.Gamma,
		Theta: result.Theta,
		Vega:  result.Vega,
		Rho:   result.Rho,
	}, nil
}

// GetLatestResult 获取最新定价结果
func (s *PricingQueryService) GetLatestResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	return s.repo.GetLatestPricingResult(ctx, symbol)
}

func (s *PricingQueryService) GetPrice(ctx context.Context, symbol string) (*PriceDTO, error) {
	price, err := s.repo.GetLatestPrice(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return s.toDTO(price), nil
}

func (s *PricingQueryService) ListPrices(ctx context.Context, symbols []string) ([]*PriceDTO, error) {
	prices, err := s.repo.ListLatestPrices(ctx, symbols)
	if err != nil {
		return nil, err
	}
	var dtos []*PriceDTO
	for _, p := range prices {
		dtos = append(dtos, s.toDTO(p))
	}
	return dtos, nil
}

func (s *PricingQueryService) toDTO(p *domain.Price) *PriceDTO {
	if p == nil {
		return nil
	}
	return &PriceDTO{
		Symbol:    p.Symbol,
		Bid:       p.Bid,
		Ask:       p.Ask,
		Mid:       p.Mid,
		Timestamp: p.Timestamp,
		Source:    p.Source,
	}
}
