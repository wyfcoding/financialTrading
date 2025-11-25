// Package grpc 包含 gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/fynnwu/FinancialTrading/go-api/market-data"
	"github.com/fynnwu/FinancialTrading/internal/market-data/application"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MarketDataHandler gRPC 处理器
type MarketDataHandler struct {
	pb.UnimplementedMarketDataServiceServer
	quoteService *application.QuoteApplicationService
}

// NewMarketDataHandler 创建 gRPC 处理器
func NewMarketDataHandler(quoteService *application.QuoteApplicationService) *MarketDataHandler {
	return &MarketDataHandler{
		quoteService: quoteService,
	}
}

// GetLatestQuote 获取最新行情
func (h *MarketDataHandler) GetLatestQuote(ctx context.Context, req *pb.GetLatestQuoteRequest) (*pb.QuoteResponse, error) {
	// 验证输入
	if req.Symbol == "" {
		logger.WithContext(ctx).Warn("Invalid request: symbol is required")
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	// 调用应用服务
	appReq := &application.GetLatestQuoteRequest{
		Symbol: req.Symbol,
	}

	quoteDTO, err := h.quoteService.GetLatestQuote(ctx, appReq)
	if err != nil {
		logger.WithContext(ctx).Error("Failed to get latest quote",
			zap.String("symbol", req.Symbol),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 转换为 protobuf 响应
	resp := &pb.QuoteResponse{
		Symbol:    quoteDTO.Symbol,
		BidPrice:  parseFloat(quoteDTO.BidPrice),
		AskPrice:  parseFloat(quoteDTO.AskPrice),
		BidSize:   parseFloat(quoteDTO.BidSize),
		AskSize:   parseFloat(quoteDTO.AskSize),
		LastPrice: parseFloat(quoteDTO.LastPrice),
		LastSize:  parseFloat(quoteDTO.LastSize),
		Timestamp: quoteDTO.Timestamp,
	}

	return resp, nil
}

// GetKlines 获取 K 线数据
func (h *MarketDataHandler) GetKlines(ctx context.Context, req *pb.GetKlinesRequest) (*pb.KlinesResponse, error) {
	// 验证输入
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}
	if req.Interval == "" {
		return nil, status.Error(codes.InvalidArgument, "interval is required")
	}

	logger.WithContext(ctx).Debug("GetKlines called",
		zap.String("symbol", req.Symbol),
		zap.String("interval", req.Interval),
		zap.Int32("limit", req.Limit),
	)

	// 返回空响应（实现待补充）
	return &pb.KlinesResponse{
		Symbol:   req.Symbol,
		Interval: req.Interval,
		Klines:   make([]*pb.Kline, 0),
	}, nil
}

// GetOrderBook 获取订单簿
func (h *MarketDataHandler) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.OrderBookResponse, error) {
	// 验证输入
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	depth := req.Depth
	if depth <= 0 {
		depth = 20
	}

	logger.WithContext(ctx).Debug("GetOrderBook called",
		zap.String("symbol", req.Symbol),
		zap.Int32("depth", depth),
	)

	// 返回空响应（实现待补充）
	return &pb.OrderBookResponse{
		Symbol:    req.Symbol,
		Bids:      make([]*pb.OrderBookLevel, 0),
		Asks:      make([]*pb.OrderBookLevel, 0),
		Timestamp: 0,
	}, nil
}

// SubscribeQuotes 订阅行情更新（流式）
func (h *MarketDataHandler) SubscribeQuotes(req *pb.SubscribeQuotesRequest, stream pb.MarketDataService_SubscribeQuotesServer) error {
	// 验证输入
	if len(req.Symbols) == 0 {
		return status.Error(codes.InvalidArgument, "symbols is required")
	}

	logger.Debug("SubscribeQuotes called",
		zap.Strings("symbols", req.Symbols),
	)

	// 实现待补充（需要实现实时推送机制）
	return nil
}

// GetTrades 获取交易历史
func (h *MarketDataHandler) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.TradesResponse, error) {
	// 验证输入
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	logger.WithContext(ctx).Debug("GetTrades called",
		zap.String("symbol", req.Symbol),
		zap.Int32("limit", limit),
	)

	// 返回空响应（实现待补充）
	return &pb.TradesResponse{
		Symbol: req.Symbol,
		Trades: make([]*pb.TradeRecord, 0),
	}, nil
}

// parseFloat 解析浮点数
func parseFloat(s string) float64 {
	// 实际应用中应使用 strconv.ParseFloat
	return 0.0
}
