package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/marketdata/domain"
)

type MarketDataApplicationService struct {
	quoteRepo domain.QuoteRepository
	tradeRepo domain.TradeRepository
}

func NewMarketDataApplicationService(quoteRepo domain.QuoteRepository, tradeRepo domain.TradeRepository) *MarketDataApplicationService {
	return &MarketDataApplicationService{
		quoteRepo: quoteRepo,
		tradeRepo: tradeRepo,
	}
}

func (s *MarketDataApplicationService) IngestQuote(ctx context.Context, cmd IngestQuoteCommand) error {
	quote := domain.NewQuote(
		cmd.Symbol,
		cmd.BidPrice,
		cmd.AskPrice,
		cmd.BidSize,
		cmd.AskSize,
		cmd.LastPrice,
		cmd.LastSize,
	)
	return s.quoteRepo.Save(ctx, quote)
}

func (s *MarketDataApplicationService) IngestTrade(ctx context.Context, cmd IngestTradeCommand) error {
	trade := &domain.Trade{
		ID:       cmd.TradeID,
		Symbol:   cmd.Symbol,
		Price:    cmd.Price,
		Quantity: cmd.Quantity,
		Side:     cmd.Side,
		// Assuming timestamp is generated or passed, simplied to Now if missing
	}
	return s.tradeRepo.Save(ctx, trade)
}
