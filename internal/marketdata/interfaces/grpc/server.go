package grpc

import (
	"context"
	"strconv"

	marketdatav1 "github.com/wyfcoding/financialtrading/go-api/marketdata/v1"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MarketDataGrpcServer struct {
	marketdatav1.UnimplementedMarketDataServiceServer
	query *application.MarketDataQueryService
}

func NewMarketDataGrpcServer(query *application.MarketDataQueryService) *MarketDataGrpcServer {
	return &MarketDataGrpcServer{query: query}
}

func (s *MarketDataGrpcServer) GetLatestQuote(ctx context.Context, req *marketdatav1.GetLatestQuoteRequest) (*marketdatav1.GetLatestQuoteResponse, error) {
	dto, err := s.query.GetLatestQuote(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dto == nil {
		return nil, status.Error(codes.NotFound, "quote not found")
	}

	return &marketdatav1.GetLatestQuoteResponse{
		Symbol:    dto.Symbol,
		BidPrice:  parseToFloat(dto.BidPrice),
		AskPrice:  parseToFloat(dto.AskPrice),
		BidSize:   parseToFloat(dto.BidSize),
		AskSize:   parseToFloat(dto.AskSize),
		LastPrice: parseToFloat(dto.LastPrice),
		LastSize:  parseToFloat(dto.LastSize),
		Timestamp: dto.Timestamp,
	}, nil
}

func (s *MarketDataGrpcServer) GetKlines(ctx context.Context, req *marketdatav1.GetKlinesRequest) (*marketdatav1.GetKlinesResponse, error) {
	dtos, err := s.query.GetKlines(ctx, req.Symbol, req.Interval, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	klines := make([]*marketdatav1.Kline, len(dtos))
	for i, d := range dtos {
		klines[i] = &marketdatav1.Kline{
			OpenTime:  d.OpenTime,
			Open:      parseToFloat(d.Open),
			High:      parseToFloat(d.High),
			Low:       parseToFloat(d.Low),
			Close:     parseToFloat(d.Close),
			Volume:    parseToFloat(d.Volume),
			CloseTime: d.CloseTime,
		}
	}
	return &marketdatav1.GetKlinesResponse{
		Symbol:   req.Symbol,
		Interval: req.Interval,
		Klines:   klines,
	}, nil
}

func (s *MarketDataGrpcServer) GetTrades(ctx context.Context, req *marketdatav1.GetTradesRequest) (*marketdatav1.GetTradesResponse, error) {
	dtos, err := s.query.GetTrades(ctx, req.Symbol, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	trades := make([]*marketdatav1.TradeRecord, len(dtos))
	for i, d := range dtos {
		trades[i] = &marketdatav1.TradeRecord{
			TradeId:   d.TradeID,
			Symbol:    d.Symbol,
			Price:     parseToFloat(d.Price),
			Quantity:  parseToFloat(d.Quantity),
			Side:      d.Side,
			Timestamp: d.Timestamp,
		}
	}
	return &marketdatav1.GetTradesResponse{Symbol: req.Symbol, Trades: trades}, nil
}

func parseToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
