package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

type ClearingQueryService struct {
	repo domain.SettlementRepository
}

func NewClearingQueryService(repo domain.SettlementRepository) *ClearingQueryService {
	return &ClearingQueryService{repo: repo}
}

func (q *ClearingQueryService) GetSettlement(ctx context.Context, id string) (*SettlementDTO, error) {
	agg, err := q.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if agg == nil {
		return nil, nil
	}

	var settledAt int64
	if agg.SettledAt != nil {
		settledAt = agg.SettledAt.Unix()
	}

	return &SettlementDTO{
		SettlementID: agg.ID,
		TradeID:      agg.TradeID,
		Status:       string(agg.Status),
		TotalAmount:  agg.TotalAmount.String(),
		SettledAt:    settledAt,
		ErrorMessage: agg.ErrorMessage,
	}, nil
}
