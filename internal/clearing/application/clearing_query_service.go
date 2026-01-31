package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
)

// SettlementDTO 结算单传输对象
type SettlementDTO struct {
	SettlementID  string
	TradeID       string
	Status        string
	TotalAmount   string
	SettledAt     int64
	ErrorMessage  string
	TradesSettled int32
	TotalTrades   int32
}

// MarginDTO 保证金传输对象
type MarginDTO struct {
	Symbol           string
	BaseMarginRate   decimal.Decimal
	VolatilityFactor decimal.Decimal
}

func (m *MarginDTO) CurrentMarginRate() decimal.Decimal {
	return m.BaseMarginRate.Mul(m.VolatilityFactor)
}

type ClearingQueryService struct {
	repo       domain.SettlementRepository
	searchRepo domain.SettlementSearchRepository
	redisRepo  domain.MarginRedisRepository
}

func NewClearingQueryService(
	repo domain.SettlementRepository,
	searchRepo domain.SettlementSearchRepository,
	redisRepo domain.MarginRedisRepository,
) *ClearingQueryService {
	return &ClearingQueryService{
		repo:       repo,
		searchRepo: searchRepo,
		redisRepo:  redisRepo,
	}
}

func (q *ClearingQueryService) GetSettlement(ctx context.Context, id string) (*SettlementDTO, error) {
	agg, err := q.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if agg == nil {
		return nil, nil
	}
	return q.toDTO(agg), nil
}

func (q *ClearingQueryService) GetClearingStatus(ctx context.Context, id string) (*SettlementDTO, error) {
	return q.GetSettlement(ctx, id)
}

func (q *ClearingQueryService) ListSettlements(ctx context.Context, userID, symbol string, limit, offset int) ([]*SettlementDTO, int64, error) {
	aggs, total, err := q.searchRepo.Search(ctx, userID, symbol, limit, offset)
	if err != nil {
		// Fallback to MySQL
		mysqlAggs, mysqlErr := q.repo.List(ctx, limit)
		if mysqlErr != nil {
			return nil, 0, mysqlErr
		}
		aggs = mysqlAggs
		total = int64(len(mysqlAggs))
	}

	dtos := make([]*SettlementDTO, 0, len(aggs))
	for _, a := range aggs {
		dtos = append(dtos, q.toDTO(a))
	}
	return dtos, total, nil
}

func (q *ClearingQueryService) GetMarginRequirement(ctx context.Context, userID, symbol string) (*MarginDTO, error) {
	// Try Redis first
	cached, _ := q.redisRepo.Get(ctx, userID)
	if cached != nil {
		// 实际上这里应该有更复杂的解包逻辑，此处仅示意
		return &MarginDTO{Symbol: symbol, BaseMarginRate: decimal.NewFromFloat(0.05)}, nil
	}

	// Fallback to static or engine
	return &MarginDTO{
		Symbol:           symbol,
		BaseMarginRate:   decimal.NewFromFloat(0.05),
		VolatilityFactor: decimal.NewFromFloat(1.1),
	}, nil
}

func (q *ClearingQueryService) toDTO(agg *domain.Settlement) *SettlementDTO {
	var settledAt int64
	if agg.SettledAt != nil {
		settledAt = agg.SettledAt.Unix()
	}
	return &SettlementDTO{
		SettlementID: agg.SettlementID,
		TradeID:      agg.TradeID,
		Status:       string(agg.Status),
		TotalAmount:  agg.TotalAmount.String(),
		SettledAt:    settledAt,
		ErrorMessage: agg.ErrorMessage,
	}
}
