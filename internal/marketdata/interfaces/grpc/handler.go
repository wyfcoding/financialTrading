// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"strconv"

	pb "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MarketDataHandler gRPC 处理器
// 负责处理与市场数据相关的 gRPC 请求
type MarketDataHandler struct {
	pb.UnimplementedMarketDataServiceServer
	service *application.MarketDataService // 市场数据应用服务门面
}

// NewMarketDataHandler 创建 gRPC 处理器实例
// service: 注入的市场数据应用服务
func NewHandler(service *application.MarketDataService) *MarketDataHandler {
	return &MarketDataHandler{
		service: service,
	}
}

// GetLatestQuote 获取最新行情
func (h *MarketDataHandler) GetLatestQuote(ctx context.Context, req *pb.GetLatestQuoteRequest) (*pb.GetLatestQuoteResponse, error) {
	// 验证输入
	if req.Symbol == "" {
		logging.Warn(ctx, "Invalid request: symbol is required")
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	appReq := &application.GetLatestQuoteRequest{
		Symbol: req.Symbol,
	}

	quoteDTO, err := h.service.GetLatestQuote(ctx, appReq)
	if err != nil {
		logging.Error(ctx, "Failed to get latest quote",
			"symbol", req.Symbol,
			"error", err,
		)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetLatestQuoteResponse{
		Symbol:    quoteDTO.Symbol,
		BidPrice:  parseFloat(quoteDTO.BidPrice),
		AskPrice:  parseFloat(quoteDTO.AskPrice),
		BidSize:   parseFloat(quoteDTO.BidSize),
		AskSize:   parseFloat(quoteDTO.AskSize),
		LastPrice: parseFloat(quoteDTO.LastPrice),
		LastSize:  parseFloat(quoteDTO.LastSize),
		Timestamp: quoteDTO.Timestamp,
	}, nil
}

// GetKlines 获取 K 线数据
func (h *MarketDataHandler) GetKlines(ctx context.Context, req *pb.GetKlinesRequest) (*pb.GetKlinesResponse, error) {
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	dtos, err := h.service.GetKlines(ctx, req.Symbol, req.Interval, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	klines := make([]*pb.Kline, len(dtos))
	for i, d := range dtos {
		klines[i] = &pb.Kline{
			OpenTime:  d.OpenTime,
			Open:      parseFloat(d.Open),
			High:      parseFloat(d.High),
			Low:       parseFloat(d.Low),
			Close:     parseFloat(d.Close),
			Volume:    parseFloat(d.Volume),
			CloseTime: d.CloseTime,
		}
	}
	return &pb.GetKlinesResponse{
		Symbol:   req.Symbol,
		Interval: req.Interval,
		Klines:   klines,
	}, nil
}

// GetOrderBook 获取订单簿
func (h *MarketDataHandler) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.GetOrderBookResponse, error) {
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	depth := req.Depth
	if depth <= 0 {
		depth = 20
	}

	// 实现待补充（查询 Repository）
	return &pb.GetOrderBookResponse{
		Symbol:    req.Symbol,
		Bids:      make([]*pb.OrderBookLevel, 0),
		Asks:      make([]*pb.OrderBookLevel, 0),
		Timestamp: 0,
	}, nil
}

// SubscribeQuotes 订阅行情更新（流式）
func (h *MarketDataHandler) SubscribeQuotes(req *pb.SubscribeQuotesRequest, stream pb.MarketDataService_SubscribeQuotesServer) error {
	if len(req.Symbols) == 0 {
		return status.Error(codes.InvalidArgument, "symbols is required")
	}
	// 实现待补充（需要实时推送机制）
	return nil
}

// GetTrades 获取交易历史
func (h *MarketDataHandler) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.GetTradesResponse, error) {
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}

	dtos, err := h.service.GetTrades(ctx, req.Symbol, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	trades := make([]*pb.TradeRecord, len(dtos))
	for i, d := range dtos {
		trades[i] = &pb.TradeRecord{
			TradeId:   d.TradeID,
			Symbol:    d.Symbol,
			Price:     parseFloat(d.Price),
			Quantity:  parseFloat(d.Quantity),
			Side:      d.Side,
			Timestamp: d.Timestamp,
		}
	}
	return &pb.GetTradesResponse{Symbol: req.Symbol, Trades: trades}, nil
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
