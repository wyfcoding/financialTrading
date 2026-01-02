package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	executionv1 "github.com/wyfcoding/financialtrading/goapi/execution/v1"
	marketdatav1 "github.com/wyfcoding/financialtrading/goapi/marketdata/v1"
	orderv1 "github.com/wyfcoding/financialtrading/goapi/order/v1"
	referencedatav1 "github.com/wyfcoding/financialtrading/goapi/referencedata/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// SORManager 负责智能订单路由 (Smart Order Routing)。
type SORManager struct {
	repo      domain.ExecutionRepository
	orderCli  orderv1.OrderServiceClient
	marketCli marketdatav1.MarketDataServiceClient
	refCli    referencedatav1.ReferenceDataServiceClient
}

// NewSORManager 构造函数。
func NewSORManager(repo domain.ExecutionRepository, orderCli orderv1.OrderServiceClient, marketCli marketdatav1.MarketDataServiceClient, refCli referencedatav1.ReferenceDataServiceClient) *SORManager {
	return &SORManager{
		repo:      repo,
		orderCli:  orderCli,
		marketCli: marketCli,
		refCli:    refCli,
	}
}

// ExecuteSOR 处理 SOR 订单执行过程。
func (m *SORManager) ExecuteSOR(ctx context.Context, req *executionv1.SubmitSOROrderRequest) (*executionv1.SubmitSOROrderResponse, error) {
	slog.Info("SOR execution started", "strategy", req.Strategy, "symbol", req.Symbol, "user_id", req.UserId)

	// 动态发现可用场所 (Venues)
	exchangesResp, err := m.refCli.ListExchanges(ctx, &referencedatav1.ListExchangesRequest{PageSize: 10})
	if err != nil {
		return nil, fmt.Errorf("failed to discover venues: %w", err)
	}

	var venues []string
	for _, ex := range exchangesResp.Exchanges {
		if ex.Status == "OPEN" || ex.Status == "ACTIVE" {
			venues = append(venues, ex.Id)
		}
	}

	if len(venues) == 0 {
		return nil, fmt.Errorf("no active venues discovered")
	}

	totalQty, err := decimal.NewFromString(req.TotalQuantity)
	if err != nil {
		return nil, fmt.Errorf("invalid total quantity: %w", err)
	}

	sorID := fmt.Sprintf("SOR-%d", idgen.GenID())

	var resp *executionv1.SubmitSOROrderResponse
	switch req.Strategy {
	case "BEST_PRICE":
		resp, err = m.handleBestPrice(ctx, sorID, req.UserId, req.Symbol, req.Side, totalQty, venues)
	case "LIQUIDITY_AGGREGATION":
		resp, err = m.handleLiquidityAggregation(ctx, sorID, req.UserId, req.Symbol, req.Side, totalQty, venues)
	default:
		return nil, fmt.Errorf("unsupported SOR strategy: %s", req.Strategy)
	}

	if err != nil {
		slog.Error("SOR execution failed", "sor_id", sorID, "error", err)
		return nil, err
	}

	return resp, nil
}

func (m *SORManager) handleBestPrice(ctx context.Context, sorID, userID, symbol, side string, qty decimal.Decimal, venues []string) (*executionv1.SubmitSOROrderResponse, error) {
	var bestVenue string
	var bestPrice decimal.Decimal

	for _, v := range venues {
		// 模拟跨场所行情获取，假设场所后的后缀表示不同流动性池
		venueSymbol := fmt.Sprintf("%s:%s", symbol, v)
		priceResp, err := m.marketCli.GetLatestQuote(ctx, &marketdatav1.GetLatestQuoteRequest{Symbol: venueSymbol})
		if err != nil {
			slog.Warn("Failed to fetch price from venue", "venue", v, "error", err)
			continue
		}

		var price decimal.Decimal
		if side == "BUY" {
			price = decimal.NewFromFloat(priceResp.AskPrice)
		} else {
			price = decimal.NewFromFloat(priceResp.BidPrice)
		}

		if bestVenue == "" {
			bestVenue = v
			bestPrice = price
		} else {
			if side == "BUY" && price.LessThan(bestPrice) {
				bestVenue = v
				bestPrice = price
			} else if side == "SELL" && price.GreaterThan(bestPrice) {
				bestVenue = v
				bestPrice = price
			}
		}
	}

	if bestVenue == "" {
		return nil, fmt.Errorf("no available liquidity found on any simulated venue")
	}

	slog.Info("SOR selected best venue", "sor_id", sorID, "venue", bestVenue, "price", bestPrice.String())

	// 向最优场所发送订单
	_, err := m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:    userID,
		Symbol:    fmt.Sprintf("%s:%s", symbol, bestVenue),
		Side:      side,
		OrderType: "MARKET",
		Quantity:  qty.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to place order on best venue %s: %w", bestVenue, err)
	}

	return &executionv1.SubmitSOROrderResponse{
		SorId:  sorID,
		Status: "COMPLETED",
	}, nil
}

func (m *SORManager) handleLiquidityAggregation(ctx context.Context, sorID, userID, symbol, side string, totalQty decimal.Decimal, venues []string) (*executionv1.SubmitSOROrderResponse, error) {
	type venueDepth struct {
		name string
		size decimal.Decimal
	}
	var availableDepths []venueDepth
	var totalAvailableSize decimal.Decimal

	// 1. 获取所有场所的真实流动性深度
	for _, v := range venues {
		venueSymbol := fmt.Sprintf("%s:%s", symbol, v)
		priceResp, err := m.marketCli.GetLatestQuote(ctx, &marketdatav1.GetLatestQuoteRequest{Symbol: venueSymbol})
		if err != nil {
			continue
		}

		var size decimal.Decimal
		if side == "BUY" {
			size = decimal.NewFromFloat(priceResp.AskSize)
		} else {
			size = decimal.NewFromFloat(priceResp.BidSize)
		}

		if size.GreaterThan(decimal.Zero) {
			availableDepths = append(availableDepths, venueDepth{v, size})
			totalAvailableSize = totalAvailableSize.Add(size)
		}
	}

	if totalAvailableSize.IsZero() {
		return nil, fmt.Errorf("no available liquidity found for aggregation")
	}

	// 2. 按比例拆分订单规模并执行
	successCount := 0
	for _, vd := range availableDepths {
		// 计算分配比例: weight = venue_size / total_size
		weight := vd.size.Div(totalAvailableSize)
		allocatedQty := totalQty.Mul(weight).Round(8) // 假设精度 8

		if allocatedQty.IsZero() {
			continue
		}

		slog.Info("SOR routing liquidity share", "sor_id", sorID, "venue", vd.name, "qty", allocatedQty.String(), "weight", weight.String())
		
		_, err := m.orderCli.CreateOrder(ctx, &orderv1.CreateOrderRequest{
			UserId:    userID,
			Symbol:    fmt.Sprintf("%s:%s", symbol, vd.name),
			Side:      side,
			OrderType: "MARKET",
			Quantity:  allocatedQty.String(),
		})
		if err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("liquidity aggregation execution failed on all venues")
	}

	return &executionv1.SubmitSOROrderResponse{
		SorId:  sorID,
		Status: "AGGREGATED_SUCCESS",
	}, nil
}
