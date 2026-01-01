// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/goapi/pricing/v1"
	"github.com/wyfcoding/financialtrading/internal/pricing/application"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler gRPC 处理器
// 负责处理与定价相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedPricingServiceServer
	app *application.PricingService // 定价应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// app: 注入的定价应用服务
func NewGRPCHandler(app *application.PricingService) *GRPCHandler {
	return &GRPCHandler{app: app}
}

// GetOptionPrice 获取期权价格
func (h *GRPCHandler) GetOptionPrice(ctx context.Context, req *pb.GetOptionPriceRequest) (*pb.GetOptionPriceResponse, error) {
	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: decimal.NewFromFloat(req.Contract.StrikePrice),
		ExpiryDate:  req.Contract.ExpiryDate.AsTime().UnixMilli(),
	}

	price, err := h.app.GetOptionPrice(ctx, contract, decimal.NewFromFloat(req.UnderlyingPrice), req.Volatility, req.RiskFreeRate)
	if err != nil {
		slog.Error("Failed to get option price", "symbol", req.Contract.Symbol, "error", err)
		return nil, err
	}

	p_val, ok := price.Float64()
	if !ok {
		slog.Warn("Failed to convert price to float64", "symbol", req.Contract.Symbol, "price", price.String())
	}
	return &pb.GetOptionPriceResponse{
		Price:           p_val,
		CalculationTime: timestamppb.Now(),
	}, nil
}

// GetGreeks 获取希腊字母
func (h *GRPCHandler) GetGreeks(ctx context.Context, req *pb.GetGreeksRequest) (*pb.GetGreeksResponse, error) {
	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: decimal.NewFromFloat(req.Contract.StrikePrice),
		ExpiryDate:  req.Contract.ExpiryDate.AsTime().UnixMilli(),
	}

	greeks, err := h.app.GetGreeks(ctx, contract, decimal.NewFromFloat(req.UnderlyingPrice), req.Volatility, req.RiskFreeRate)
	if err != nil {
		slog.Error("Failed to get greeks", "symbol", req.Contract.Symbol, "error", err)
		return nil, err
	}

	d_val, ok1 := greeks.Delta.Float64()
	g_val, ok2 := greeks.Gamma.Float64()
	t_val, ok3 := greeks.Theta.Float64()
	v_val, ok4 := greeks.Vega.Float64()
	r_val, ok5 := greeks.Rho.Float64()
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		slog.Warn("Failed to convert some greeks to float64", "symbol", req.Contract.Symbol, "ok1", ok1, "ok2", ok2, "ok3", ok3, "ok4", ok4, "ok5", ok5)
	}

	return &pb.GetGreeksResponse{
		Greeks: &pb.Greeks{
			Delta: d_val,
			Gamma: g_val,
			Theta: t_val,
			Vega:  v_val,
			Rho:   r_val,
		},
		CalculationTime: timestamppb.Now(),
	}, nil
}
