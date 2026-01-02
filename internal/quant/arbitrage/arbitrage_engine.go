package arbitrage

import (
	"context"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	marketdatav1 "github.com/wyfcoding/financialtrading/goapi/marketdata/v1"
)

// ArbitrageOpportunity 发现的套利机会
type ArbitrageOpportunity struct {
	Symbol      string
	BuyVenue    string
	SellVenue   string
	Spread      decimal.Decimal
	MaxQuantity int64
}

// ArbitrageEngine 跨市场套利引擎
type ArbitrageEngine struct {
	marketCli marketdatav1.MarketDataServiceClient
	analyzer  *LiquidityAnalyzer
}

func NewArbitrageEngine(marketCli marketdatav1.MarketDataServiceClient) *ArbitrageEngine {
	return &ArbitrageEngine{
		marketCli: marketCli,
		analyzer:  NewLiquidityAnalyzer(),
	}
}

// FindOpportunities 在多个场所间寻找指定交易对的套利机会
func (e *ArbitrageEngine) FindOpportunities(ctx context.Context, symbol string, venues []string) ([]ArbitrageOpportunity, error) {
	var mu sync.Mutex
	var opportunities []ArbitrageOpportunity
	var wg sync.WaitGroup

	type venueQuote struct {
		venue string
		quote *marketdatav1.GetLatestQuoteResponse
	}
	quotes := make([]venueQuote, 0, len(venues))

	// 1. 并发获取各场所报价
	for _, v := range venues {
		wg.Add(1)
		go func(venue string) {
			defer wg.Done()
			venueSymbol := fmt.Sprintf("%s:%s", symbol, venue)
			resp, err := e.marketCli.GetLatestQuote(ctx, &marketdatav1.GetLatestQuoteRequest{Symbol: venueSymbol})
			if err == nil {
				mu.Lock()
				quotes = append(quotes, venueQuote{venue, resp})
				mu.Unlock()
			}
		}(v)
	}
	wg.Wait()

	if len(quotes) < 2 {
		return nil, nil
	}

	// 2. 交叉对比寻找价差
	for i := 0; i < len(quotes); i++ {
		for j := 0; j < len(quotes); j++ {
			if i == j {
				continue
			}

			askI := decimal.NewFromFloat(quotes[i].quote.AskPrice) // 场所 I 的买入成本 (Ask)
			bidJ := decimal.NewFromFloat(quotes[j].quote.BidPrice) // 场所 J 的卖出收益 (Bid)

			// 如果 卖出价 > 买入价，存在正向套利空间
			if bidJ.GreaterThan(askI) {
				spread := bidJ.Sub(askI)

				// 3. 计算该路径上的最大可承载深度 (利用已有的 Dinic 分析器)
				// 这里的 MaxAmount 简化为两个市场挂单量的最小值
				sizeI := int64(quotes[i].quote.AskSize)
				sizeJ := int64(quotes[j].quote.BidSize)

				liquidity := []MarketLiquidity{
					{FromAsset: quotes[i].venue, ToAsset: quotes[j].venue, MaxAmount: sizeI},
					{FromAsset: quotes[i].venue, ToAsset: quotes[j].venue, MaxAmount: sizeJ},
				}

				maxQty := e.analyzer.CalculateMaxRouteAmount(quotes[i].venue, quotes[j].venue, liquidity)

				opportunities = append(opportunities, ArbitrageOpportunity{
					Symbol:      symbol,
					BuyVenue:    quotes[i].venue,
					SellVenue:   quotes[j].venue,
					Spread:      spread,
					MaxQuantity: maxQty,
				})
			}
		}
	}

	return opportunities, nil
}
