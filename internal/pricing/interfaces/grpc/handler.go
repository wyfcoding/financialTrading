// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/pricing/v1"
	"github.com/wyfcoding/financialTrading/internal/pricing/application"
	"github.com/wyfcoding/financialTrading/internal/pricing/domain"
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
// 处理 gRPC GetOptionPrice 请求
func (h *GRPCHandler) GetOptionPrice(ctx context.Context, req *pb.GetOptionPriceRequest) (*pb.GetOptionPriceResponse, error) {
	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: req.Contract.StrikePrice,
		ExpiryDate:  req.Contract.ExpiryDate.AsTime(),
	}

	price, err := h.app.GetOptionPrice(ctx, contract, req.UnderlyingPrice, req.Volatility, req.RiskFreeRate)
	if err != nil {
		return nil, err
	}

	return &pb.GetOptionPriceResponse{
		Price:           price,
		CalculationTime: timestamppb.Now(),
	}, nil
}

// GetGreeks 获取希腊字母
func (h *GRPCHandler) GetGreeks(ctx context.Context, req *pb.GetGreeksRequest) (*pb.GetGreeksResponse, error) {
	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: req.Contract.StrikePrice,
		ExpiryDate:  req.Contract.ExpiryDate.AsTime(),
	}

	greeks, err := h.app.GetGreeks(ctx, contract, req.UnderlyingPrice, req.Volatility, req.RiskFreeRate)
	if err != nil {
		return nil, err
	}

	return &pb.GetGreeksResponse{
		Greeks: &pb.Greeks{
			Delta: greeks.Delta,
			Gamma: greeks.Gamma,
			Theta: greeks.Theta,
			Vega:  greeks.Vega,
			Rho:   greeks.Rho,
		},
		CalculationTime: timestamppb.Now(),
	}, nil
}
