// 包  gRPC 处理器（Handler）的实现。
// 这一层是接口层（Interfaces Layer）的一部分，负责适配外部的 gRPC 请求，
// 并将其转换为对应用层（Application Layer）的调用。
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler 是清算服务的 gRPC 处理器。
type Handler struct {
	pb.UnimplementedClearingServiceServer
	app *application.ClearingService
}

// NewHandler 创建 gRPC 处理器实例。
func NewHandler(app *application.ClearingService) *Handler {
	return &Handler{
		app: app,
	}
}

// SettleTrade 结算单笔交易
func (h *Handler) SettleTrade(ctx context.Context, req *pb.SettleTradeRequest) (*pb.SettleTradeResponse, error) {
	start := time.Now()
	slog.Info("gRPC SettleTrade received", "trade_id", req.TradeId, "symbol", req.Symbol)

	qty, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)

	appReq := &application.SettleTradeRequest{
		TradeID:    req.TradeId,
		BuyUserID:  req.BuyUserId,
		SellUserID: req.SellUserId,
		Symbol:     req.Symbol,
		Quantity:   qty,
		Price:      price,
	}

	dto, err := h.app.Command.SettleTrade(ctx, appReq)
	if err != nil {
		slog.Error("gRPC SettleTrade failed", "trade_id", req.TradeId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to settle trade: %v", err)
	}

	return &pb.SettleTradeResponse{
		SettlementId:   dto.SettlementID,
		TradeId:        dto.TradeID,
		Status:         dto.Status,
		SettlementTime: dto.SettledAt,
		ErrorMessage:   dto.ErrorMessage,
	}, nil
}

// GetSettlements 获取清算记录
func (h *Handler) GetSettlements(ctx context.Context, req *pb.GetSettlementsRequest) (*pb.GetSettlementsResponse, error) {
	start := time.Now()
	slog.Info("gRPC GetSettlements received", "user_id", req.UserId)

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	dtos, _, err := h.app.Query.ListSettlements(ctx, req.UserId, "", limit, 0)
	if err != nil {
		slog.Error("gRPC GetSettlements failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get settlements: %v", err)
	}

	records := make([]*pb.Settlement, 0, len(dtos))
	for _, dto := range dtos {
		records = append(records, &pb.Settlement{
			SettlementId:   dto.SettlementID,
			TradeId:        dto.TradeID,
			Status:         dto.Status,
			SettlementTime: dto.SettledAt,
		})
	}

	return &pb.GetSettlementsResponse{
		Settlements: records,
	}, nil
}

// SagaMarkSettlementCompleted Saga 正向: 确认结算成功
func (h *Handler) SagaMarkSettlementCompleted(ctx context.Context, req *pb.SagaSettlementRequest) (*pb.SagaSettlementResponse, error) {
	if err := h.app.Command.SagaMarkSettlementCompleted(ctx, req.SettlementId); err != nil {
		return nil, status.Errorf(codes.Internal, "SagaMarkSettlementCompleted failed: %v", err)
	}
	return &pb.SagaSettlementResponse{Success: true}, nil
}

// SagaMarkSettlementFailed Saga 补偿: 标记结算失败
func (h *Handler) SagaMarkSettlementFailed(ctx context.Context, req *pb.SagaSettlementRequest) (*pb.SagaSettlementResponse, error) {
	if err := h.app.Command.SagaMarkSettlementFailed(ctx, req.SettlementId, req.Reason); err != nil {
		return nil, status.Errorf(codes.Internal, "SagaMarkSettlementFailed failed: %v", err)
	}
	return &pb.SagaSettlementResponse{Success: true}, nil
}

// ExecuteEODClearing 执行日终清算
func (h *Handler) ExecuteEODClearing(ctx context.Context, req *pb.ExecuteEODClearingRequest) (*pb.ExecuteEODClearingResponse, error) {
	clearingID, err := h.app.Command.ExecuteEODClearing(ctx, req.ClearingDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to execute EOD clearing: %v", err)
	}

	return &pb.ExecuteEODClearingResponse{
		Status:     "PROCESSING",
		ClearingId: clearingID,
		StartTime:  time.Now().Unix(),
	}, nil
}

// GetClearingStatus 获取状态
func (h *Handler) GetClearingStatus(ctx context.Context, req *pb.GetClearingStatusRequest) (*pb.GetClearingStatusResponse, error) {
	dto, err := h.app.Query.GetClearingStatus(ctx, req.ClearingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get clearing status: %v", err)
	}
	if dto == nil {
		return nil, status.Errorf(codes.NotFound, "clearing task with id '%s' not found", req.ClearingId)
	}

	return &pb.GetClearingStatusResponse{
		ClearingId:      dto.SettlementID,
		Status:          dto.Status,
		TradesProcessed: int64(dto.TradesSettled),
		TradesTotal:     int64(dto.TotalTrades),
	}, nil
}

// GetMarginRequirement 获取保证金
func (h *Handler) GetMarginRequirement(ctx context.Context, req *pb.GetMarginRequirementRequest) (*pb.GetMarginRequirementResponse, error) {
	// 获取保证金要求
	margin, err := h.app.Query.GetMarginRequirement(ctx, "", req.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get margin requirement: %v", err)
	}

	return &pb.GetMarginRequirementResponse{
		Symbol:               margin.Symbol,
		BaseMarginRate:       margin.BaseMarginRate.String(),
		VolatilityMultiplier: margin.VolatilityFactor.String(),
		CurrentMarginRate:    margin.CurrentMarginRate().String(),
	}, nil
}
