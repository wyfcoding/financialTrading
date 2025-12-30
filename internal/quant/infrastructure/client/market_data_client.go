package client

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	market_data "github.com/wyfcoding/financialtrading/goapi/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/quant/domain"
	"github.com/wyfcoding/pkg/grpcclient"
	"github.com/wyfcoding/pkg/logging"
	"google.golang.org/grpc"
)

// MarketDataClientImpl 市场数据服务客户端实现
type MarketDataClientImpl struct {
	client market_data.MarketDataServiceClient
}

// NewMarketDataClient 创建市场数据服务客户端
func NewMarketDataClient(target string) (domain.MarketDataClient, error) {
	conn, err := grpcclient.NewClientFactory(logging.Default()).NewClient(target)
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

// GetHistoricalData 获取历史价格数据
func (c *MarketDataClientImpl) GetHistoricalData(ctx context.Context, symbol string, start, end int64) ([]decimal.Decimal, error) {
	// 注意：当前 proto 不支持 StartTime/EndTime 过滤，仅支持 limit
	req := &market_data.GetKlinesRequest{
		Symbol:   symbol,
		Interval: "1d", // 默认日线
		Limit:    500,  // 获取较多数据
	}

	resp, err := c.client.GetKlines(ctx, req)
	if err != nil {
		logging.Error(ctx, "Failed to get klines",
			"symbol", symbol,
			"error", err,
		)
		return nil, err
	}

	prices := make([]decimal.Decimal, len(resp.Klines))
	for i, k := range resp.Klines {
		// 使用 proto 定义的 close 字段
		prices[i] = decimal.NewFromFloat(k.Close)
	}

	logging.Info(ctx, "Retrieved historical data",
		"symbol", symbol,
		"count", len(prices),
	)

	return prices, nil
}
