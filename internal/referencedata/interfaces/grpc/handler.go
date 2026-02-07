package grpc

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/referencedata/v1"
	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与参考数据相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedReferenceDataServiceServer
	cmd   *application.ReferenceDataCommandService
	query *application.ReferenceDataQueryService
}

// NewHandler 创建 gRPC 处理器实例
func NewHandler(cmd *application.ReferenceDataCommandService, query *application.ReferenceDataQueryService) *Handler {
	return &Handler{
		cmd:   cmd,
		query: query,
	}
}

// GetInstrument 获取合约详情
func (h *Handler) GetInstrument(ctx context.Context, req *pb.GetInstrumentRequest) (*pb.GetInstrumentResponse, error) {
	dto, err := h.query.GetInstrument(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dto == nil {
		return &pb.GetInstrumentResponse{}, nil
	}
	return &pb.GetInstrumentResponse{Instrument: h.toProtoInstrument(dto)}, nil
}

// ListInstruments 列出合约详情
func (h *Handler) ListInstruments(ctx context.Context, _ *pb.ListInstrumentsRequest) (*pb.ListInstrumentsResponse, error) {
	limit := 100
	instruments, err := h.query.ListInstruments(ctx, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	protoInstruments := make([]*pb.Instrument, 0, len(instruments))
	for _, i := range instruments {
		protoInstruments = append(protoInstruments, h.toProtoInstrument(i))
	}
	return &pb.ListInstrumentsResponse{Instruments: protoInstruments}, nil
}

// GetSymbol 获取交易对详情
func (h *Handler) GetSymbol(ctx context.Context, req *pb.GetSymbolRequest) (*pb.GetSymbolResponse, error) {
	id := req.Id
	if id == "" {
		id = req.SymbolCode
	}
	symbol, err := h.query.GetSymbol(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if symbol == nil {
		return &pb.GetSymbolResponse{}, nil
	}
	return &pb.GetSymbolResponse{Symbol: h.toProtoSymbol(symbol)}, nil
}

// ListSymbols 列出交易对
func (h *Handler) ListSymbols(ctx context.Context, req *pb.ListSymbolsRequest) (*pb.ListSymbolsResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	symbols, err := h.query.ListSymbols(ctx, req.ExchangeId, req.Status, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	protoSymbols := make([]*pb.Symbol, 0, len(symbols))
	for _, s := range symbols {
		protoSymbols = append(protoSymbols, h.toProtoSymbol(s))
	}
	return &pb.ListSymbolsResponse{Symbols: protoSymbols}, nil
}

// GetExchange 获取交易所信息
func (h *Handler) GetExchange(ctx context.Context, req *pb.GetExchangeRequest) (*pb.GetExchangeResponse, error) {
	id := req.Id
	if id == "" {
		id = req.Name
	}
	exchange, err := h.query.GetExchange(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if exchange == nil {
		return &pb.GetExchangeResponse{}, nil
	}
	return &pb.GetExchangeResponse{Exchange: h.toProtoExchange(exchange)}, nil
}

// ListExchanges 列出交易所
func (h *Handler) ListExchanges(ctx context.Context, req *pb.ListExchangesRequest) (*pb.ListExchangesResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	exchanges, err := h.query.ListExchanges(ctx, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	protoExchanges := make([]*pb.Exchange, 0, len(exchanges))
	for _, e := range exchanges {
		protoExchanges = append(protoExchanges, h.toProtoExchange(e))
	}
	return &pb.ListExchangesResponse{Exchanges: protoExchanges}, nil
}

func (h *Handler) toProtoInstrument(d *application.InstrumentDTO) *pb.Instrument {
	if d == nil {
		return nil
	}
	tick, _ := decimal.NewFromString(d.TickSize)
	lot, _ := decimal.NewFromString(d.LotSize)
	return &pb.Instrument{
		Symbol:        d.Symbol,
		BaseCurrency:  d.BaseCurrency,
		QuoteCurrency: d.QuoteCurrency,
		TickSize:      tick.InexactFloat64(),
		LotSize:       lot.InexactFloat64(),
		Type:          mapInstrumentType(d.Type),
		MaxLeverage:   int32(d.MaxLeverage),
	}
}

func mapInstrumentType(t string) pb.InstrumentType {
	switch t {
	case "SPOT":
		return pb.InstrumentType_SPOT
	case "FUTURE":
		return pb.InstrumentType_FUTURE
	case "OPTION":
		return pb.InstrumentType_OPTION
	default:
		return pb.InstrumentType_INSTRUMENT_TYPE_UNSPECIFIED
	}
}

func (h *Handler) toProtoSymbol(s *application.SymbolDTO) *pb.Symbol {
	if s == nil {
		return nil
	}
	minOrder, _ := decimal.NewFromString(s.MinOrderSize)
	pricePrec, _ := decimal.NewFromString(s.PricePrecision)
	return &pb.Symbol{
		Id:             s.ID,
		BaseCurrency:   s.BaseCurrency,
		QuoteCurrency:  s.QuoteCurrency,
		ExchangeId:     s.ExchangeID,
		SymbolCode:     s.SymbolCode,
		Status:         s.Status,
		MinOrderSize:   minOrder.InexactFloat64(),
		PricePrecision: pricePrec.InexactFloat64(),
		CreatedAt:      timestamppb.New(time.Unix(s.CreatedAt, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(s.UpdatedAt, 0)),
	}
}

func (h *Handler) toProtoExchange(e *application.ExchangeDTO) *pb.Exchange {
	if e == nil {
		return nil
	}
	return &pb.Exchange{
		Id:        e.ID,
		Name:      e.Name,
		Country:   e.Country,
		Status:    e.Status,
		Timezone:  e.Timezone,
		CreatedAt: timestamppb.New(time.Unix(e.CreatedAt, 0)),
		UpdatedAt: timestamppb.New(time.Unix(e.UpdatedAt, 0)),
	}
}
