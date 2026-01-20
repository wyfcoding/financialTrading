package client

import (
	"context"

	"github.com/shopspring/decimal"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
)

type GRPCMarketDataClient struct {
	client marketdatav1.MarketDataServiceClient
}

func NewGRPCMarketDataClient(client marketdatav1.MarketDataServiceClient) *GRPCMarketDataClient {
	return &GRPCMarketDataClient{client: client}
}

func (c *GRPCMarketDataClient) GetVolatility(ctx context.Context, symbol string) (decimal.Decimal, error) {
	resp, err := c.client.GetVolatility(ctx, &marketdatav1.GetVolatilityRequest{
		Symbol: symbol,
	})
	if err != nil {
		return decimal.Zero, err
	}

	return decimal.NewFromFloat(resp.Volatility), nil
}
