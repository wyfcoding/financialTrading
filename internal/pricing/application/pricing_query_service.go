package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/algorithm/finance"
)

// PricingQueryService 处理所有定价相关的查询操作（Queries）。
type PricingQueryService struct {
	repo     domain.PricingRepository
	readRepo domain.PricingReadRepository
	bsCalc   *finance.BlackScholesCalculator
}

// NewPricingQueryService 构造函数。
func NewPricingQueryService(repo domain.PricingRepository, readRepo domain.PricingReadRepository) *PricingQueryService {
	return &PricingQueryService{
		repo:     repo,
		readRepo: readRepo,
		bsCalc:   finance.NewBlackScholesCalculator(),
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

	res, calcErr := s.bsCalc.Calculate(
		string(contract.Type),
		underlyingPrice,
		contract.StrikePrice,
		decimal.NewFromFloat(timeToExpiry),
		decimal.NewFromFloat(riskFreeRate),
		decimal.NewFromFloat(volatility),
		decimal.Zero, // Default div to 0 if not provided in Query
	)
	if calcErr != nil {
		return nil, calcErr
	}

	return &domain.Greeks{
		Delta: res.Delta,
		Gamma: res.Gamma,
		Theta: res.Theta,
		Vega:  res.Vega,
		Rho:   res.Rho,
	}, nil
}

// GetLatestResult 获取最新定价结果
func (s *PricingQueryService) GetLatestResult(ctx context.Context, symbol string) (*domain.PricingResult, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetLatestPricingResult(ctx, symbol); err == nil && cached != nil {
			return cached, nil
		}
	}

	res, err := s.repo.GetLatestPricingResult(ctx, symbol)
	if err != nil || res == nil {
		return res, err
	}
	if s.readRepo != nil {
		_ = s.readRepo.SavePricingResult(ctx, res)
	}
	return res, nil
}

func (s *PricingQueryService) GetPrice(ctx context.Context, symbol string) (*PriceDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.GetLatestPrice(ctx, symbol); err == nil && cached != nil {
			return s.toDTO(cached), nil
		}
	}

	price, err := s.repo.GetLatestPrice(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if price != nil && s.readRepo != nil {
		_ = s.readRepo.SavePrice(ctx, price)
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
		if p != nil && s.readRepo != nil {
			_ = s.readRepo.SavePrice(ctx, p)
		}
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
