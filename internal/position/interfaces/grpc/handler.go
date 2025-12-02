// Package grpc 包含 gRPC 处理器实现
package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialTrading/go-api/position"
	"github.com/wyfcoding/financialTrading/internal/position/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与持仓管理相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedPositionServiceServer
	appService *application.PositionApplicationService // 持仓应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的持仓应用服务
func NewGRPCHandler(appService *application.PositionApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// GetPositions 获取持仓列表
// 处理 gRPC GetPositions 请求
func (h *GRPCHandler) GetPositions(ctx context.Context, req *pb.GetPositionsRequest) (*pb.PositionsResponse, error) {
	// 解析分页参数
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int((req.Page - 1) * req.PageSize)
	if offset < 0 {
		offset = 0
	}

	dtos, total, err := h.appService.GetPositions(ctx, req.UserId, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get positions: %v", err)
	}

	pbPositions := make([]*pb.PositionResponse, 0, len(dtos))
	for _, dto := range dtos {
		pbPositions = append(pbPositions, h.toProtoResponse(dto))
	}

	return &pb.PositionsResponse{
		Positions: pbPositions,
		Total:     total,
	}, nil
}

// GetPosition 获取持仓详情
// 处理 gRPC GetPosition 请求
func (h *GRPCHandler) GetPosition(ctx context.Context, req *pb.GetPositionRequest) (*pb.PositionResponse, error) {
	dto, err := h.appService.GetPosition(ctx, req.PositionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get position: %v", err)
	}

	return h.toProtoResponse(dto), nil
}

// ClosePosition 平仓
// 处理 gRPC ClosePosition 请求
func (h *GRPCHandler) ClosePosition(ctx context.Context, req *pb.ClosePositionRequest) (*pb.PositionResponse, error) {
	closePrice, err := decimal.NewFromString(req.ClosePrice)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid close price: %v", err)
	}

	err = h.appService.ClosePosition(ctx, req.PositionId, closePrice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close position: %v", err)
	}

	// Fetch updated position to return
	dto, err := h.appService.GetPosition(ctx, req.PositionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated position: %v", err)
	}

	return h.toProtoResponse(dto), nil
}

func (h *GRPCHandler) toProtoResponse(dto *application.PositionDTO) *pb.PositionResponse {
	var closedAt int64
	if dto.ClosedAt != nil {
		closedAt = *dto.ClosedAt
	}

	return &pb.PositionResponse{
		PositionId:    dto.PositionID,
		UserId:        dto.UserID,
		Symbol:        dto.Symbol,
		Side:          dto.Side,
		Quantity:      dto.Quantity,
		EntryPrice:    dto.EntryPrice,
		CurrentPrice:  dto.CurrentPrice,
		UnrealizedPnl: dto.UnrealizedPnL,
		RealizedPnl:   dto.RealizedPnL,
		OpenedAt:      dto.OpenedAt,
		ClosedAt:      closedAt,
	}
}
