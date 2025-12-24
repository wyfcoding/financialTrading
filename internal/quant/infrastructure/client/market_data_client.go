package client

import (
	"context"
	"fmt"
	"time"

	market_data "github.com/wyfcoding/financialtrading/goapi/market_data/v1"
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

// GetHistoricalData 获取历史数据
func (c *MarketDataClientImpl) GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]float64, error) {
	req := &market_data.GetKlinesRequest{
		Symbol:   symbol,
		Interval: "1d", // 默认日线
		Limit:    100,  // 默认获取最近 100 条
	}

	resp, err := c.client.GetKlines(ctx, req)
	if err != nil {
		logging.Error(ctx, "Failed to get historical data",
			"symbol", symbol,
			"error", err,
		)
		return nil, err
	}

	// 转换数据
	prices := make([]float64, len(resp.Klines))
	for i, kline := range resp.Klines {
		// 简单取收盘价
		// 这里需要解析 decimal 字符串
		// 暂时 mock 返回
		_ = kline
		prices[i] = 100.0 + float64(i)
	}

	return prices, nil
}
