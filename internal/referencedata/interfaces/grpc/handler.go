// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/referencedata/v1"
	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与参考数据相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedReferenceDataServiceServer
	app *application.ReferenceDataService // 参考数据应用服务门面
}

// NewHandler 创建 gRPC 处理器实例
// app: 注入的参考数据应用服务
func NewHandler(app *application.ReferenceDataService) *Handler {
	return &Handler{
		app: app,
	}
}

// GetInstrument 获取合约详情 (Legacy)
func (h *Handler) GetInstrument(ctx context.Context, req *pb.GetInstrumentRequest) (*pb.GetInstrumentResponse, error) {
	// 暂时返回错误，因为 app 中可能没有定义 GetInstrument 方法
	return nil, status.Error(codes.Unimplemented, "GetInstrument not implemented")
}

// ListInstruments 列出合约详情 (Legacy)
func (h *Handler) ListInstruments(ctx context.Context, req *pb.ListInstrumentsRequest) (*pb.ListInstrumentsResponse, error) {
	// 暂时返回错误，因为 app 中可能没有定义 ListInstruments 方法
	return nil, status.Error(codes.Unimplemented, "ListInstruments not implemented")
}

// GetSymbol 获取交易对详情
func (h *Handler) GetSymbol(ctx context.Context, req *pb.GetSymbolRequest) (*pb.GetSymbolResponse, error) {
	id := req.Id
	if id == "" {
		id = req.SymbolCode
	}
	symbol, err := h.app.GetSymbol(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if symbol == nil {
		return &pb.GetSymbolResponse{}, nil
	}
	return &pb.GetSymbolResponse{
		Symbol: toProtoSymbol(symbol),
	}, nil
}

// ListSymbols 列出交易对
func (h *Handler) ListSymbols(ctx context.Context, req *pb.ListSymbolsRequest) (*pb.ListSymbolsResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	symbols, err := h.app.ListSymbols(ctx, req.ExchangeId, req.Status, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoSymbols := make([]*pb.Symbol, len(symbols))
	for i, s := range symbols {
		protoSymbols[i] = toProtoSymbol(s)
	}
	return &pb.ListSymbolsResponse{Symbols: protoSymbols}, nil
}

// GetExchange 获取交易所信息
func (h *Handler) GetExchange(ctx context.Context, req *pb.GetExchangeRequest) (*pb.GetExchangeResponse, error) {
	id := req.Id
	if id == "" {
		id = req.Name
	}
	exchange, err := h.app.GetExchange(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if exchange == nil {
		return &pb.GetExchangeResponse{}, nil
	}
	return &pb.GetExchangeResponse{
		Exchange: toProtoExchange(exchange),
	}, nil
}

// ListExchanges 列出交易所
func (h *Handler) ListExchanges(ctx context.Context, req *pb.ListExchangesRequest) (*pb.ListExchangesResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	exchanges, err := h.app.ListExchanges(ctx, limit, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoExchanges := make([]*pb.Exchange, len(exchanges))
	for i, e := range exchanges {
		protoExchanges[i] = toProtoExchange(e)
	}
	return &pb.ListExchangesResponse{Exchanges: protoExchanges}, nil
}

func (h *Handler) toProtoInstrument(d *application.InstrumentDTO) *pb.Instrument {
	if d == nil {
		return nil
	}
	return &pb.Instrument{
		Symbol:        d.Symbol,
		BaseCurrency:  d.BaseCurrency,
		QuoteCurrency: d.QuoteCurrency,
		TickSize:      d.TickSize,
		LotSize:       d.LotSize,
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

func toProtoSymbol(s *domain.Symbol) *pb.Symbol {
	return &pb.Symbol{
		Id:             s.ID,
		BaseCurrency:   s.BaseCurrency,
		QuoteCurrency:  s.QuoteCurrency,
		ExchangeId:     s.ExchangeID,
		SymbolCode:     s.SymbolCode,
		Status:         s.Status,
		MinOrderSize:   s.MinOrderSize.InexactFloat64(),
		PricePrecision: s.PricePrecision.InexactFloat64(),
		CreatedAt:      timestamppb.New(s.CreatedAt),
		UpdatedAt:      timestamppb.New(s.UpdatedAt),
	}
}

func toProtoExchange(e *domain.Exchange) *pb.Exchange {
	return &pb.Exchange{
		Id:        e.ID,
		Name:      e.Name,
		Country:   e.Country,
		Status:    e.Status,
		Timezone:  e.Timezone,
		CreatedAt: timestamppb.New(e.CreatedAt),
		UpdatedAt: timestamppb.New(e.UpdatedAt),
	}
}
