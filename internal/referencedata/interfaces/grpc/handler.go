// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/referencedata/v1"
	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"github.com/wyfcoding/financialtrading/internal/referencedata/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与参考数据相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedReferenceDataServiceServer
	app *application.ReferenceDataService // 参考数据应用服务
}

// NewHandler 创建 gRPC 处理器实例
// app: 注入的参考数据应用服务
func NewHandler(app *application.ReferenceDataService) *Handler {
	return &Handler{
		app: app,
	}
}

// GetSymbol 获取交易对
// 处理 gRPC GetSymbol 请求
func (h *Handler) GetSymbol(ctx context.Context, req *pb.GetSymbolRequest) (*pb.GetSymbolResponse, error) {
	start := time.Now()
	id := req.Id
	if id == "" {
		id = req.SymbolCode
	}
	slog.Debug("gRPC GetSymbol received", "id", id)

	symbol, err := h.app.GetSymbol(ctx, id)
	if err != nil {
		slog.Error("gRPC GetSymbol failed", "id", id, "error", err, "duration", time.Since(start))
		return nil, err
	}
	if symbol == nil {
		slog.Debug("gRPC GetSymbol successful (not found)", "id", id, "duration", time.Since(start))
		return &pb.GetSymbolResponse{}, nil
	}

	slog.Debug("gRPC GetSymbol successful", "id", id, "duration", time.Since(start))
	return &pb.GetSymbolResponse{
		Symbol: toProtoSymbol(symbol),
	}, nil
}

// ListSymbols 列出交易对
func (h *Handler) ListSymbols(ctx context.Context, req *pb.ListSymbolsRequest) (*pb.ListSymbolsResponse, error) {
	start := time.Now()
	slog.Debug("gRPC ListSymbols received", "exchange_id", req.ExchangeId, "status", req.Status)

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	offset := 0 // 简单分页，实际应解析 page_token

	symbols, err := h.app.ListSymbols(ctx, req.ExchangeId, req.Status, limit, offset)
	if err != nil {
		slog.Error("gRPC ListSymbols failed", "exchange_id", req.ExchangeId, "error", err, "duration", time.Since(start))
		return nil, err
	}

	protoSymbols := make([]*pb.Symbol, len(symbols))
	for i, s := range symbols {
		protoSymbols[i] = toProtoSymbol(s)
	}

	slog.Debug("gRPC ListSymbols successful", "exchange_id", req.ExchangeId, "count", len(protoSymbols), "duration", time.Since(start))
	return &pb.ListSymbolsResponse{
		Symbols: protoSymbols,
	}, nil
}

// GetExchange 获取交易所
func (h *Handler) GetExchange(ctx context.Context, req *pb.GetExchangeRequest) (*pb.GetExchangeResponse, error) {
	start := time.Now()
	id := req.Id
	if id == "" {
		id = req.Name
	}
	slog.Debug("gRPC GetExchange received", "id", id)

	exchange, err := h.app.GetExchange(ctx, id)
	if err != nil {
		slog.Error("gRPC GetExchange failed", "id", id, "error", err, "duration", time.Since(start))
		return nil, err
	}
	if exchange == nil {
		slog.Debug("gRPC GetExchange successful (not found)", "id", id, "duration", time.Since(start))
		return &pb.GetExchangeResponse{}, nil
	}

	slog.Debug("gRPC GetExchange successful", "id", id, "duration", time.Since(start))
	return &pb.GetExchangeResponse{
		Exchange: toProtoExchange(exchange),
	}, nil
}

// ListExchanges 列出交易所
func (h *Handler) ListExchanges(ctx context.Context, req *pb.ListExchangesRequest) (*pb.ListExchangesResponse, error) {
	start := time.Now()
	slog.Debug("gRPC ListExchanges received")

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	offset := 0 // 简单分页

	exchanges, err := h.app.ListExchanges(ctx, limit, offset)
	if err != nil {
		slog.Error("gRPC ListExchanges failed", "error", err, "duration", time.Since(start))
		return nil, err
	}

	protoExchanges := make([]*pb.Exchange, len(exchanges))
	for i, e := range exchanges {
		protoExchanges[i] = toProtoExchange(e)
	}

	slog.Debug("gRPC ListExchanges successful", "count", len(protoExchanges), "duration", time.Since(start))
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
