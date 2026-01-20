package client

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	market_data "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/domain"
	"github.com/wyfcoding/pkg/config"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/logging"
	"github.com/wyfcoding/pkg/metrics"
	"google.golang.org/grpc"
)

// MarketDataClientImpl 市场数据服务客户端实现
type MarketDataClientImpl struct {
	client market_data.MarketDataServiceClient
}

// NewMarketDataClient 创建市场数据服务客户端
func NewMarketDataClient(target string, m *metrics.Metrics, cbCfg config.CircuitBreakerConfig) (domain.MarketDataClient, error) {
	conn, err := grpcclient.NewClientFactory(logging.Default(), m, cbCfg).NewClient(target)
	if err != nil {
		return nil, fmt.Errorf("failed to create market data service client: %w", err)
	}

	return &MarketDataClientImpl{
		client: market_data.NewMarketDataServiceClient(conn),
	}, nil
}

// NewMarketDataClientFromConn 从现有连接创建客户端
func NewMarketDataClientFromConn(conn *grpc.ClientConn) domain.MarketDataClient {
	return &MarketDataClientImpl{
		client: market_data.NewMarketDataServiceClient(conn),
	}
}

// GetPrice 获取最新价格
func (c *MarketDataClientImpl) GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	req := &market_data.GetLatestQuoteRequest{Symbol: symbol}
	resp, err := c.client.GetLatestQuote(ctx, req)
	if err != nil {
		logging.Error(ctx, "Failed to get latest quote",
			"symbol", symbol,
			"error", err,
		)
		return decimal.Zero, err
	}

	// 从响应中获取最新价格
	price := decimal.NewFromFloat(resp.GetLastPrice())
	if price.IsZero() {
		// 如果没有最新成交价，使用买卖中间价
		bidPrice := decimal.NewFromFloat(resp.GetBidPrice())
		askPrice := decimal.NewFromFloat(resp.GetAskPrice())
		if bidPrice.IsPositive() && askPrice.IsPositive() {
			price = bidPrice.Add(askPrice).Div(decimal.NewFromInt(2))
		}
	}

	logging.Info(ctx, "Retrieved latest price from market data",
		"symbol", symbol,
		"price", price,
	)

	return price, nil
}
