package client

import (
	"context"
	"fmt"
	"time"

	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
)

type gRPCMarketDataClient struct {
	client marketdatav1.MarketDataServiceClient
}

func NewMarketDataClient(client marketdatav1.MarketDataServiceClient) domain.MarketDataClient {
	return &gRPCMarketDataClient{client: client}
}

func (c *gRPCMarketDataClient) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is empty")
	}
	resp, err := c.client.GetLatestQuote(ctx, &marketdatav1.GetLatestQuoteRequest{
		Symbol: symbol,
	})
	if err != nil {
		return nil, err
	}
	quote := &domain.Quote{
		Symbol:    resp.Symbol,
		BidPrice:  resp.BidPrice,
		AskPrice:  resp.AskPrice,
		LastPrice: resp.LastPrice,
		UpdatedAt: time.Unix(resp.Timestamp, 0),
	}
	return quote, nil
}
