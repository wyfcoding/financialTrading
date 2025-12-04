package client

import (
	"context"
	"fmt"

	market_data "github.com/wyfcoding/financialTrading/go-api/market-data"
	"github.com/wyfcoding/financialTrading/internal/market-making/domain"
	"github.com/wyfcoding/financialTrading/pkg/grpcclient"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// MarketDataClientImpl 市场数据服务客户端实现
type MarketDataClientImpl struct {
	client market_data.MarketDataServiceClient
}

// NewMarketDataClient 创建市场数据服务客户端
func NewMarketDataClient(cfg grpcclient.ClientConfig) (domain.MarketDataClient, error) {
	conn, err := grpcclient.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create market data service client: %w", err)
	}

	return &MarketDataClientImpl{
		client: market_data.NewMarketDataServiceClient(conn),
	}, nil
}

// GetPrice 获取最新价格
func (c *MarketDataClientImpl) GetPrice(ctx context.Context, symbol string) (float64, error) {
	req := &market_data.GetLatestQuoteRequest{
		Symbol: symbol,
	}

	resp, err := c.client.GetLatestQuote(ctx, req)
	if err != nil {
		logger.Error(ctx, "Failed to get latest quote",
			"symbol", symbol,
			"error", err,
		)
		return 0, err
	}

	// TODO: Parse decimal from string
	_ = resp
	return 100.0, nil
}
