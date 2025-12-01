// Package grpc 包含 gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/clearing"
	"github.com/wyfcoding/financialTrading/internal/clearing/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与清算相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedClearingServiceServer
	appService *application.ClearingApplicationService // 清算应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的清算应用服务
func NewGRPCHandler(appService *application.ClearingApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// SettleTrade 清算交易
// 处理 gRPC SettleTrade 请求
func (h *GRPCHandler) SettleTrade(ctx context.Context, req *pb.SettleTradeRequest) (*pb.SettlementResponse, error) {
	// 调用应用服务进行清算
	err := h.appService.SettleTrade(ctx, &application.SettleTradeRequest{
		TradeID:    req.TradeId,
		BuyUserID:  req.BuyUserId,
		SellUserID: req.SellUserId,
		Symbol:     req.Symbol,
		Quantity:   req.Quantity,
		Price:      req.Price,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to settle trade: %v", err)
	}

	return &pb.SettlementResponse{
		TradeId: req.TradeId,
		Status:  "COMPLETED",
		// SettlementID and SettlementTime would ideally be returned by the service
	}, nil
}

func (h *GRPCHandler) ExecuteEODClearing(ctx context.Context, req *pb.ExecuteEODClearingRequest) (*pb.EODClearingResponse, error) {
	err := h.appService.ExecuteEODClearing(ctx, req.ClearingDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to execute EOD clearing: %v", err)
	}

	return &pb.EODClearingResponse{
		Status: "PROCESSING",
		// ClearingID would ideally be returned by the service
	}, nil
}

func (h *GRPCHandler) GetClearingStatus(ctx context.Context, req *pb.GetClearingStatusRequest) (*pb.ClearingStatusResponse, error) {
	clearing, err := h.appService.GetClearingStatus(ctx, req.ClearingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get clearing status: %v", err)
	}
	if clearing == nil {
		return nil, status.Errorf(codes.NotFound, "clearing not found")
	}

	return &pb.ClearingStatusResponse{
		ClearingId:      clearing.ClearingID,
		Status:          clearing.Status,
		TradesProcessed: int64(clearing.TradesSettled),
		TradesTotal:     int64(clearing.TotalTrades),
	}, nil
}
