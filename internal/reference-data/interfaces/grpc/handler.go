// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/reference-data"
	"github.com/wyfcoding/financialTrading/internal/reference-data/application"
	"github.com/wyfcoding/financialTrading/internal/reference-data/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与参考数据相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedReferenceDataServiceServer
	app *application.ReferenceDataService // 参考数据应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的参考数据应用服务
func NewGRPCHandler(app *application.ReferenceDataService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// GetSymbol 获取交易对
// 处理 gRPC GetSymbol 请求
func (h *GRPCHandler) GetSymbol(ctx context.Context, req *pb.GetSymbolRequest) (*pb.GetSymbolResponse, error) {
	id := req.Id
	if id == "" {
		id = req.SymbolCode
	}

	symbol, err := h.app.GetSymbol(ctx, id)
	if err != nil {
		return nil, err
	}
	if symbol == nil {
		return &pb.GetSymbolResponse{}, nil
	}

	return &pb.GetSymbolResponse{
		Symbol: toProtoSymbol(symbol),
	}, nil
}

// ListSymbols 列出交易对
func (h *GRPCHandler) ListSymbols(ctx context.Context, req *pb.ListSymbolsRequest) (*pb.ListSymbolsResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	offset := 0 // 简单分页，实际应解析 page_token

	symbols, err := h.app.ListSymbols(ctx, req.ExchangeId, req.Status, limit, offset)
	if err != nil {
		return nil, err
	}

	protoSymbols := make([]*pb.Symbol, len(symbols))
	for i, s := range symbols {
		protoSymbols[i] = toProtoSymbol(s)
	}

	return &pb.ListSymbolsResponse{
		Symbols: protoSymbols,
	}, nil
}

// GetExchange 获取交易所
func (h *GRPCHandler) GetExchange(ctx context.Context, req *pb.GetExchangeRequest) (*pb.GetExchangeResponse, error) {
	id := req.Id
	if id == "" {
		id = req.Name
	}

	exchange, err := h.app.GetExchange(ctx, id)
	if err != nil {
		return nil, err
	}
	if exchange == nil {
		return &pb.GetExchangeResponse{}, nil
	}

	return &pb.GetExchangeResponse{
		Exchange: toProtoExchange(exchange),
	}, nil
}

// ListExchanges 列出交易所
func (h *GRPCHandler) ListExchanges(ctx context.Context, req *pb.ListExchangesRequest) (*pb.ListExchangesResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	offset := 0 // 简单分页

	exchanges, err := h.app.ListExchanges(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	protoExchanges := make([]*pb.Exchange, len(exchanges))
	for i, e := range exchanges {
		protoExchanges[i] = toProtoExchange(e)
	}

	return &pb.ListExchangesResponse{
		Exchanges: protoExchanges,
	}, nil
}

func toProtoSymbol(s *domain.Symbol) *pb.Symbol {
	return &pb.Symbol{
		Id:             s.ID,
		BaseCurrency:   s.BaseCurrency,
		QuoteCurrency:  s.QuoteCurrency,
		ExchangeId:     s.ExchangeID,
		SymbolCode:     s.SymbolCode,
		Status:         s.Status,
		MinOrderSize:   s.MinOrderSize,
		PricePrecision: s.PricePrecision,
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
