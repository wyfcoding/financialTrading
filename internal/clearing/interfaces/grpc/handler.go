// 包  gRPC 处理器（Handler）的实现。
// 这一层是接口层（Interfaces Layer）的一部分，负责适配外部的 gRPC 请求，
// 并将其转换为对应用层（Application Layer）的调用。
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/goapi/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler 是清算服务的 gRPC 处理器。
// 它实现了由 apoch 生成的 `ClearingServiceServer` 接口。
// 其核心职责是作为 gRPC 协议与内部应用逻辑之间的桥梁。
type GRPCHandler struct {
	pb.UnimplementedClearingServiceServer                              // 嵌入未实现的 apoch 服务，确保向前兼容
	service                               *application.ClearingService // 依赖注入的应用服务实例
}

// 创建 gRPC 处理器实例。
//
// @param service 注入的清算应用服务实例。
// @return *GRPCHandler 返回一个新的 gRPC 处理器实例。
func NewGRPCHandler(service *application.ClearingService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// SettleTrade 实现了 gRPC 的 SettleTrade 方法。
// 它接收 gRPC 请求，将其转换为应用层的 DTO，然后调用应用服务来处理。
func (h *GRPCHandler) SettleTrade(ctx context.Context, req *pb.SettleTradeRequest) (*pb.SettleTradeResponse, error) {
	start := time.Now()
	slog.Info("gRPC SettleTrade received", "trade_id", req.TradeId, "buy_user_id", req.BuyUserId, "sell_user_id", req.SellUserId, "symbol", req.Symbol)

	// 1. 将 gRPC 请求对象 (*pb.SettleTradeRequest) 转换为应用层 DTO (*application.SettleTradeRequest)。
	appReq := &application.SettleTradeRequest{
		TradeID:    req.TradeId,
		BuyUserID:  req.BuyUserId,
		SellUserID: req.SellUserId,
		Symbol:     req.Symbol,
		Quantity:   req.Quantity,
		Price:      req.Price,
	}

	// 2. 调用应用服务来执行核心业务逻辑,接收返回的 settlementID。
	settlementID, err := h.service.SettleTrade(ctx, appReq)
	if err != nil {
		slog.Error("gRPC SettleTrade failed", "trade_id", req.TradeId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to settle trade: %v", err)
	}

	slog.Info("gRPC SettleTrade successful", "trade_id", req.TradeId, "settlement_id", settlementID, "duration", time.Since(start))
	// 4. 构建并返回 gRPC 响应,填充从应用服务返回的 settlementID。
	return &pb.SettleTradeResponse{
		TradeId:        req.TradeId,
		Status:         "COMPLETED", // 假设状态为已完成
		SettlementId:   settlementID,
		SettlementTime: time.Now().Unix(),
	}, nil
}

// ExecuteEODClearing 实现了 gRPC 的 ExecuteEODClearing 方法。
func (h *GRPCHandler) ExecuteEODClearing(ctx context.Context, req *pb.ExecuteEODClearingRequest) (*pb.ExecuteEODClearingResponse, error) {
	start := time.Now()
	slog.Info("gRPC ExecuteEODClearing received", "clearing_date", req.ClearingDate)

	// 调用应用服务启动日终清算流程,接收返回的 clearingID。
	clearingID, err := h.service.ExecuteEODClearing(ctx, req.ClearingDate)
	if err != nil {
		slog.Error("gRPC ExecuteEODClearing failed", "clearing_date", req.ClearingDate, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to execute EOD clearing: %v", err)
	}

	slog.Info("gRPC ExecuteEODClearing successful", "clearing_date", req.ClearingDate, "clearing_id", clearingID, "duration", time.Since(start))
	// 返回包含 clearingID 的响应,表示任务已开始。
	return &pb.ExecuteEODClearingResponse{
		Status:     "PROCESSING", // 表示任务已开始处理
		ClearingId: clearingID,
		StartTime:  time.Now().Unix(),
	}, nil
}

func (h *GRPCHandler) GetClearingStatus(ctx context.Context, req *pb.GetClearingStatusRequest) (*pb.GetClearingStatusResponse, error) {
	slog.Debug("gRPC GetClearingStatus received", "clearing_id", req.ClearingId)

	// 调用应用服务获取清算任务状态。
	clearing, err := h.service.GetClearingStatus(ctx, req.ClearingId)
	if err != nil {
		slog.Error("gRPC GetClearingStatus failed", "clearing_id", req.ClearingId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get clearing status: %v", err)
	}
	// 如果应用服务返回 nil（表示未找到），则返回 gRPC 的 NotFound 错误。
	if clearing == nil {
		slog.Warn("gRPC GetClearingStatus task not found", "clearing_id", req.ClearingId)
		return nil, status.Errorf(codes.NotFound, "clearing task with id '%s' not found", req.ClearingId)
	}

	// 计算完成百分比
	var completionPercentage float32
	if clearing.TotalTrades > 0 {
		completionPercentage = float32(clearing.TradesSettled) / float32(clearing.TotalTrades) * 100
	}

	slog.Debug("gRPC GetClearingStatus successful", "clearing_id", req.ClearingId, "status", clearing.Status)
	// 将从应用层获取的领域对象转换为 gRPC 响应对象。
	return &pb.GetClearingStatusResponse{
		ClearingId:         clearing.ClearingID,
		Status:             clearing.Status,
		TradesProcessed:    clearing.TradesSettled,
		TradesTotal:        clearing.TotalTrades,
		ProgressPercentage: int64(completionPercentage),
	}, nil
}

func (h *GRPCHandler) GetMarginRequirement(ctx context.Context, req *pb.GetMarginRequirementRequest) (*pb.GetMarginRequirementResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetMarginRequirement received", "symbol", req.Symbol)

	margin, err := h.service.GetMarginRequirement(ctx, req.Symbol)
	if err != nil {
		slog.Error("gRPC GetMarginRequirement failed", "symbol", req.Symbol, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get margin requirement: %v", err)
	}

	slog.Debug("gRPC GetMarginRequirement successful", "symbol", req.Symbol, "duration", time.Since(start))
	return &pb.GetMarginRequirementResponse{
		Symbol:               margin.Symbol,
		BaseMarginRate:       margin.BaseMarginRate.String(),
		VolatilityMultiplier: margin.VolatilityFactor.String(),
		CurrentMarginRate:    margin.CurrentMarginRate().String(),
	}, nil
}
