package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
)

type ReferenceDataApplicationService struct {
	repo domain.ReferenceRepository
}

func NewReferenceDataApplicationService(repo domain.ReferenceRepository) *ReferenceDataApplicationService {
	return &ReferenceDataApplicationService{repo: repo}
}

func (s *ReferenceDataApplicationService) GetInstrument(ctx context.Context, symbol string) (*InstrumentDTO, error) {
	instr, err := s.repo.GetInstrument(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return s.toDTO(instr), nil
}

func (s *ReferenceDataApplicationService) ListInstruments(ctx context.Context) ([]*InstrumentDTO, error) {
	instruments, err := s.repo.ListInstruments(ctx)
	if err != nil {
		return nil, err
	}
	var dtos []*InstrumentDTO
	for _, i := range instruments {
		dtos = append(dtos, s.toDTO(i))
	}
	return dtos, nil
}

func (s *ReferenceDataApplicationService) toDTO(i *domain.Instrument) *InstrumentDTO {
	return &InstrumentDTO{
		Symbol:        i.Symbol,
		BaseCurrency:  i.BaseCurrency,
		QuoteCurrency: i.QuoteCurrency,
		TickSize:      i.TickSize,
		LotSize:       i.LotSize,
		Type:          string(i.Type),
		MaxLeverage:   i.MaxLeverage,
	}
}
