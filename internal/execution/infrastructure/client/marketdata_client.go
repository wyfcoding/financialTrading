package client

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
)

type GRPCMarketDataProvider struct {
	client marketdatav1.MarketDataServiceClient
}

func NewGRPCMarketDataProvider(client marketdatav1.MarketDataServiceClient) *GRPCMarketDataProvider {
	return &GRPCMarketDataProvider{client: client}
}

func (p *GRPCMarketDataProvider) GetVenueDepths(ctx context.Context, symbol string) ([]*domain.VenueDepth, error) {
	resp, err := p.client.GetOrderBook(ctx, &marketdatav1.GetOrderBookRequest{
		Symbol: symbol,
		Depth:  20,
	})
	if err != nil {
		return nil, err
	}

	// 映射到 domain.VenueDepth
	// 由于目前简化模型，假设只有一个主要 Venue
	depth := &domain.VenueDepth{
		VenueID: "MAIN",
		Symbol:  symbol,
	}

	for _, ask := range resp.Asks {
		p := decimal.NewFromFloat(ask.Price)
		q := decimal.NewFromFloat(ask.Quantity)
		depth.Asks = append(depth.Asks, domain.PriceLevel{Price: p, Quantity: q})
	}
	for _, bid := range resp.Bids {
		p := decimal.NewFromFloat(bid.Price)
		q := decimal.NewFromFloat(bid.Quantity)
		depth.Bids = append(depth.Bids, domain.PriceLevel{Price: p, Quantity: q})
	}

	return []*domain.VenueDepth{depth}, nil
}

func (p *GRPCMarketDataProvider) GetRecentVolume(ctx context.Context, symbol string, duration time.Duration) (decimal.Decimal, error) {
	// 简化处理：从 MarketData 服务获取最近 1 分钟的 K 线成交量
	resp, err := p.client.GetKlines(ctx, &marketdatav1.GetKlinesRequest{
		Symbol:   symbol,
		Interval: "1m",
		Limit:    1,
	})
	if err != nil {
		return decimal.Zero, err
	}

	if len(resp.Klines) == 0 {
		return decimal.Zero, nil
	}

	return decimal.NewFromFloat(resp.Klines[0].Volume), nil
}
