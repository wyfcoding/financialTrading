package client

import (
	"context"
	"fmt"

	market_data "github.com/wyfcoding/financialtrading/goapi/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/logging"
	"google.golang.org/grpc"
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

// NewMarketDataClientFromConn 从现有连接创建客户端
func NewMarketDataClientFromConn(conn *grpc.ClientConn) domain.MarketDataClient {
	return &MarketDataClientImpl{
		client: market_data.NewMarketDataServiceClient(conn),
	}
}

// GetPrice 获取最新价格
func (c *MarketDataClientImpl) GetPrice(ctx context.Context, symbol string) (float64, error) {
	req := &market_data.GetLatestQuoteRequest{
		Symbol: symbol,
	}

	resp, err := c.client.GetLatestQuote(ctx, req)
	if err != nil {
		logging.Error(ctx, "Failed to get latest quote",
			"symbol", symbol,
			"error", err,
		)
		return 0, err
	}

	// 从响应中获取最新价格
	// protobuf 已定义为 float64，直接返回
	price := resp.GetLastPrice()
	if price == 0 {
		// 如果没有最新成交价，使用中间价
		bidPrice := resp.GetBidPrice()
		askPrice := resp.GetAskPrice()
		if bidPrice > 0 && askPrice > 0 {
			price = (bidPrice + askPrice) / 2
		}
	}

	logging.Info(ctx, "Retrieved latest price from market data",
		"symbol", symbol,
		"price", price,
	)

	return price, nil
}
