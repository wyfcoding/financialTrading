package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type MarketDataQueryService struct {
	quoteRepo domain.QuoteRepository
	klineRepo domain.KlineRepository
	tradeRepo domain.TradeRepository
}

func NewMarketDataQueryService(
	quoteRepo domain.QuoteRepository,
	klineRepo domain.KlineRepository,
	tradeRepo domain.TradeRepository,
) *MarketDataQueryService {
	return &MarketDataQueryService{
		quoteRepo: quoteRepo,
		klineRepo: klineRepo,
		tradeRepo: tradeRepo,
	}
}

func (q *MarketDataQueryService) GetLatestQuote(ctx context.Context, symbol string) (*QuoteDTO, error) {
	quote, err := q.quoteRepo.GetLatest(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, nil
	}

	return &QuoteDTO{
		Symbol:    quote.Symbol,
		BidPrice:  quote.BidPrice.String(),
		AskPrice:  quote.AskPrice.String(),
		BidSize:   quote.BidSize.String(),
		AskSize:   quote.AskSize.String(),
		LastPrice: quote.LastPrice.String(),
		LastSize:  quote.LastSize.String(),
		Timestamp: quote.Timestamp.UnixMilli(),
	}, nil
}

func (q *MarketDataQueryService) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*KlineDTO, error) {
	klines, err := q.klineRepo.GetKlines(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]*KlineDTO, len(klines))
	for i, k := range klines {
		dtos[i] = &KlineDTO{
			OpenTime:  k.OpenTime.UnixMilli(),
			Open:      k.Open.String(),
			High:      k.High.String(),
			Low:       k.Low.String(),
			Close:     k.Close.String(),
			Volume:    k.Volume.String(),
			CloseTime: k.CloseTime.UnixMilli(),
		}
	}
	return dtos, nil
}

func (q *MarketDataQueryService) GetTrades(ctx context.Context, symbol string, limit int) ([]*TradeDTO, error) {
	trades, err := q.tradeRepo.GetTrades(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
	dtos := make([]*TradeDTO, len(trades))
	for i, t := range trades {
		dtos[i] = &TradeDTO{
			TradeID:   t.ID,
			Symbol:    t.Symbol,
			Price:     t.Price.String(),
			Quantity:  t.Quantity.String(),
			Side:      t.Side,
			Timestamp: t.Timestamp.UnixMilli(),
		}
	}
	return dtos, nil
}
