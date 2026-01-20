package grpc

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClearingGrpcServer struct {
	clearingv1.UnimplementedClearingServiceServer
	appService   *application.ClearingService
	queryService *application.ClearingQueryService
}

func NewClearingGrpcServer(
	appService *application.ClearingService,
	queryService *application.ClearingQueryService,
) *ClearingGrpcServer {
	return &ClearingGrpcServer{
		appService:   appService,
		queryService: queryService,
	}
}

// SettleTrade 结算单笔交易
func (s *ClearingGrpcServer) SettleTrade(ctx context.Context, req *clearingv1.SettleTradeRequest) (*clearingv1.SettleTradeResponse, error) {
	qty, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)

	cmd := application.SettleTradeCommand{
		TradeID:    req.TradeId,
		BuyUserID:  req.BuyUserId,
		SellUserID: req.SellUserId,
		Symbol:     req.Symbol,
		Quantity:   qty,
		Price:      price,
	}

	dto, err := s.appService.SettleTrade(ctx, &cmd)
	if err != nil {
		slog.ErrorContext(ctx, "settle trade failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &clearingv1.SettleTradeResponse{
		SettlementId:   dto.SettlementID,
		TradeId:        dto.TradeID,
		Status:         dto.Status,
		SettlementTime: dto.SettledAt,
		ErrorMessage:   dto.ErrorMessage,
	}, nil
}

// GetSettlements 获取清算记录 (简化版仅支持单个 ID 查)
func (s *ClearingGrpcServer) GetSettlements(ctx context.Context, req *clearingv1.GetSettlementsRequest) (*clearingv1.GetSettlementsResponse, error) {
	// 这里演示用途，假设 req 包含 settlement_id (实际 proto 定义是 user_id + limit)
	// 由于 QueryService 只实现了 GetSettlement(id)，这里暂时返回空或 Unimplemented
	// 实际应完善 QueryService.GetByUserID
	return nil, status.Error(codes.Unimplemented, "list settlements by user_id not implemented yet, use specific ID query")
}

// SagaMarkSettlementCompleted 回调
func (s *ClearingGrpcServer) SagaMarkSettlementCompleted(ctx context.Context, req *clearingv1.SagaSettlementRequest) (*clearingv1.SagaSettlementResponse, error) {
	err := s.appService.SagaMarkSettlementCompleted(ctx, req.SettlementId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &clearingv1.SagaSettlementResponse{Success: true}, nil
}

// SagaMarkSettlementFailed 补偿回调
func (s *ClearingGrpcServer) SagaMarkSettlementFailed(ctx context.Context, req *clearingv1.SagaSettlementRequest) (*clearingv1.SagaSettlementResponse, error) {
	err := s.appService.SagaMarkSettlementFailed(ctx, req.SettlementId, req.Reason)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &clearingv1.SagaSettlementResponse{Success: true}, nil
}

// ExecuteEODClearing 执行日终清算
func (s *ClearingGrpcServer) ExecuteEODClearing(ctx context.Context, req *clearingv1.ExecuteEODClearingRequest) (*clearingv1.ExecuteEODClearingResponse, error) {
	id, err := s.appService.ExecuteEODClearing(ctx, req.ClearingDate)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &clearingv1.ExecuteEODClearingResponse{
		ClearingId: id,
		Status:     "PENDING",
		StartTime:  0,
	}, nil
}

// GetClearingStatus 获取状态
func (s *ClearingGrpcServer) GetClearingStatus(ctx context.Context, req *clearingv1.GetClearingStatusRequest) (*clearingv1.GetClearingStatusResponse, error) {
	dto, err := s.appService.GetClearingStatus(ctx, req.ClearingId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if dto == nil {
		return nil, status.Error(codes.NotFound, "clearing task not found")
	}
	return &clearingv1.GetClearingStatusResponse{
		ClearingId:      dto.SettlementID,
		Status:          dto.Status,
		TradesProcessed: dto.TradesSettled,
		TradesTotal:     dto.TotalTrades,
	}, nil
}

// GetMarginRequirement 获取保证金
func (s *ClearingGrpcServer) GetMarginRequirement(ctx context.Context, req *clearingv1.GetMarginRequirementRequest) (*clearingv1.GetMarginRequirementResponse, error) {
	dto, err := s.appService.GetMarginRequirement(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &clearingv1.GetMarginRequirementResponse{
		Symbol:               dto.Symbol,
		BaseMarginRate:       dto.BaseMarginRate.String(),
		VolatilityMultiplier: dto.VolatilityFactor.String(),
		CurrentMarginRate:    dto.CurrentMarginRate().String(),
	}, nil
}
