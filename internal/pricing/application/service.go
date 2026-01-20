package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
)

type PricingApplicationService struct {
	repo domain.PriceRepository
}

func NewPricingApplicationService(repo domain.PriceRepository) *PricingApplicationService {
	return &PricingApplicationService{repo: repo}
}

func (s *PricingApplicationService) OnQuoteReceived(ctx context.Context, symbol string, bid, ask float64, source string) error {
	price := domain.NewPrice(symbol, bid, ask, source)
	return s.repo.Save(ctx, price)
}

func (s *PricingApplicationService) GetPrice(ctx context.Context, symbol string) (*PriceDTO, error) {
	price, err := s.repo.GetLatest(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return s.toDTO(price), nil
}

func (s *PricingApplicationService) ListPrices(ctx context.Context, symbols []string) ([]*PriceDTO, error) {
	prices, err := s.repo.ListLatest(ctx, symbols)
	if err != nil {
		return nil, err
	}
	var dtos []*PriceDTO
	for _, p := range prices {
		dtos = append(dtos, s.toDTO(p))
	}
	return dtos, nil
}

func (s *PricingApplicationService) toDTO(p *domain.Price) *PriceDTO {
	return &PriceDTO{
		Symbol:    p.Symbol,
		Bid:       p.Bid,
		Ask:       p.Ask,
		Mid:       p.Mid,
		Timestamp: p.Timestamp,
		Source:    p.Source,
	}
}
