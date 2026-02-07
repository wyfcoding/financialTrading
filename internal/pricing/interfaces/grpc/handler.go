package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/pricing/v1"
	"github.com/wyfcoding/financialtrading/internal/pricing/application"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler gRPC 处理器
// 负责处理与定价相关的 gRPC 请求
type Handler struct {
	pb.UnimplementedPricingServiceServer
	cmd   *application.PricingCommandService
	query *application.PricingQueryService
}

// NewHandler 创建 gRPC 处理器实例
func NewHandler(cmd *application.PricingCommandService, query *application.PricingQueryService) *Handler {
	return &Handler{cmd: cmd, query: query}
}

// GetPrice 获取单品种价格
func (h *Handler) GetPrice(ctx context.Context, req *pb.GetPriceRequest) (*pb.GetPriceResponse, error) {
	dto, err := h.query.GetPrice(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetPriceResponse{Price: h.toProto(dto)}, nil
}

// ListPrices 批量获取价格
func (h *Handler) ListPrices(ctx context.Context, req *pb.ListPricesRequest) (*pb.ListPricesResponse, error) {
	dtos, err := h.query.ListPrices(ctx, req.Symbols)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbPrices := make([]*pb.Price, 0, len(dtos))
	for _, d := range dtos {
		pbPrices = append(pbPrices, h.toProto(d))
	}
	return &pb.ListPricesResponse{Prices: pbPrices}, nil
}

// SubscribePrices 定价订阅
func (h *Handler) SubscribePrices(req *pb.SubscribePricesRequest, stream pb.PricingService_SubscribePricesServer) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			dtos, err := h.query.ListPrices(context.Background(), req.Symbols)
			if err != nil {
				continue
			}
			for _, d := range dtos {
				if err := stream.Send(&pb.PriceUpdate{Price: h.toProto(d)}); err != nil {
					return err
				}
			}
		}
	}
}

// GetOptionPrice 获取期权价格
func (h *Handler) GetOptionPrice(ctx context.Context, req *pb.GetOptionPriceRequest) (*pb.GetOptionPriceResponse, error) {
	cmd := application.PriceOptionCommand{
		Symbol:          req.Contract.Symbol,
		OptionType:      req.Contract.Type,
		StrikePrice:     req.Contract.StrikePrice,
		ExpiryDate:      req.Contract.ExpiryDate.AsTime().UnixMilli(),
		UnderlyingPrice: req.UnderlyingPrice,
		Volatility:      req.Volatility,
		RiskFreeRate:    req.RiskFreeRate,
		DividendYield:   0,
		PricingModel:    "BlackScholes",
	}

	result, err := h.cmd.PriceOption(ctx, cmd)
	if err != nil {
		slog.Error("Failed to get option price", "symbol", req.Contract.Symbol, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pVal, _ := result.OptionPrice.Float64()
	return &pb.GetOptionPriceResponse{
		Price:           pVal,
		CalculationTime: timestamppb.Now(),
	}, nil
}

// GetGreeks 获取希腊字母
func (h *Handler) GetGreeks(ctx context.Context, req *pb.GetGreeksRequest) (*pb.GetGreeksResponse, error) {
	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: decimal.NewFromFloat(req.Contract.StrikePrice),
		ExpiryDate:  req.Contract.ExpiryDate.AsTime().UnixMilli(),
	}

	greeks, err := h.query.GetGreeks(ctx, contract, decimal.NewFromFloat(req.UnderlyingPrice), req.Volatility, req.RiskFreeRate)
	if err != nil {
		slog.Error("Failed to get greeks", "symbol", req.Contract.Symbol, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	dVal, _ := greeks.Delta.Float64()
	gVal, _ := greeks.Gamma.Float64()
	tVal, _ := greeks.Theta.Float64()
	vVal, _ := greeks.Vega.Float64()
	rVal, _ := greeks.Rho.Float64()

	return &pb.GetGreeksResponse{
		Greeks: &pb.Greeks{
			Delta: dVal,
			Gamma: gVal,
			Theta: tVal,
			Vega:  vVal,
			Rho:   rVal,
		},
		CalculationTime: timestamppb.Now(),
	}, nil
}

func (h *Handler) toProto(d *application.PriceDTO) *pb.Price {
	if d == nil {
		return nil
	}
	return &pb.Price{
		Symbol:    d.Symbol,
		Bid:       d.Bid,
		Ask:       d.Ask,
		Mid:       d.Mid,
		Source:    d.Source,
		Timestamp: timestamppb.New(d.Timestamp),
	}
}
