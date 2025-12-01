// Package grpc 包含 gRPC 处理器实现
package grpc

import (
	"context"
	"fmt"
	"strconv"

	pb "github.com/wyfcoding/financialTrading/go-api/market-data"
	"github.com/wyfcoding/financialTrading/internal/market-data/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MarketDataHandler gRPC 处理器
// 负责处理与市场数据相关的 gRPC 请求
type MarketDataHandler struct {
	pb.UnimplementedMarketDataServiceServer
	quoteService *application.QuoteApplicationService // 行情应用服务
}

// NewMarketDataHandler 创建 gRPC 处理器实例
// quoteService: 注入的行情应用服务
func NewMarketDataHandler(quoteService *application.QuoteApplicationService) *MarketDataHandler {
	return &MarketDataHandler{
		quoteService: quoteService,
	}
}

// GetLatestQuote 获取最新行情
func (h *MarketDataHandler) GetLatestQuote(ctx context.Context, req *pb.GetLatestQuoteRequest) (*pb.QuoteResponse, error) {
	// 验证输入
	if req.Symbol == "" {
		logger.Warn(ctx, "Invalid request: symbol is required")
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	// 调用应用服务
	appReq := &application.GetLatestQuoteRequest{
		Symbol: req.Symbol,
	}

	quoteDTO, err := h.quoteService.GetLatestQuote(ctx, appReq)
	if err != nil {
		logger.Error(ctx, "Failed to get latest quote",
			"symbol", req.Symbol,
			"error", err,
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

	logger.Debug(ctx, "GetKlines called",
		"symbol", req.Symbol,
		"interval", req.Interval,
		"limit", req.Limit,
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

	logger.Debug(ctx, "GetOrderBook called",
		"symbol", req.Symbol,
		"depth", depth,
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

	// Note: stream.Context() is available
	ctx := stream.Context()
	logger.Debug(ctx, "SubscribeQuotes called",
		"symbols", fmt.Sprintf("%v", req.Symbols),
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

	logger.Debug(ctx, "GetTrades called",
		"symbol", req.Symbol,
		"limit", limit,
	)

	// 返回空响应（实现待补充）
	return &pb.TradesResponse{
		Symbol: req.Symbol,
		Trades: make([]*pb.TradeRecord, 0),
	}, nil
}

// parseFloat 解析浮点数
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
