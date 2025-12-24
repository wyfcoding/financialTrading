// 包  gRPC 处理器实现
package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialTrading/go-api/position/v1"
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
func (h *GRPCHandler) GetPositions(ctx context.Context, req *pb.GetPositionsRequest) (*pb.GetPositionsResponse, error) {
	// 解析分页参数
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := max(int((req.Page-1)*req.PageSize), 0)

	dtos, total, err := h.appService.GetPositions(ctx, req.UserId, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get positions: %v", err)
	}

	pbPositions := make([]*pb.Position, 0, len(dtos))
	for _, dto := range dtos {
		pbPositions = append(pbPositions, h.toProtoPosition(dto))
	}

	return &pb.GetPositionsResponse{
		Positions: pbPositions,
		Total:     total,
	}, nil
}

// GetPosition 获取持仓详情
// 处理 gRPC GetPosition 请求
func (h *GRPCHandler) GetPosition(ctx context.Context, req *pb.GetPositionRequest) (*pb.GetPositionResponse, error) {
	dto, err := h.appService.GetPosition(ctx, req.PositionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get position: %v", err)
	}

	return &pb.GetPositionResponse{
		Position: h.toProtoPosition(dto),
	}, nil
}

// ClosePosition 平仓
// 处理 gRPC ClosePosition 请求
func (h *GRPCHandler) ClosePosition(ctx context.Context, req *pb.ClosePositionRequest) (*pb.ClosePositionResponse, error) {
	closePrice, err := decimal.NewFromString(req.ClosePrice)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid close price: %v", err)
	}

	err = h.appService.ClosePosition(ctx, req.PositionId, closePrice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close position: %v", err)
	}

	// 获取更新后的头寸以返回
	dto, err := h.appService.GetPosition(ctx, req.PositionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated position: %v", err)
	}

	return &pb.ClosePositionResponse{
		Position: h.toProtoPosition(dto),
	}, nil
}

func (h *GRPCHandler) toProtoPosition(dto *application.PositionDTO) *pb.Position {
	var closedAt int64
	if dto.ClosedAt != nil {
		closedAt = *dto.ClosedAt
	}

	return &pb.Position{
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
