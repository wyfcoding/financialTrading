// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/goapi/position/v1"
	"github.com/wyfcoding/financialtrading/internal/position/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与持仓管理相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedPositionServiceServer
	service *application.PositionService // 持仓应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// service: 注入的持仓应用服务
func NewGRPCHandler(service *application.PositionService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// GetPositions 获取持仓列表
// 处理 gRPC GetPositions 请求
func (h *GRPCHandler) GetPositions(ctx context.Context, req *pb.GetPositionsRequest) (*pb.GetPositionsResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetPositions received", "user_id", req.UserId, "page", req.Page, "page_size", req.PageSize)

	// 解析分页参数
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := max(int((req.Page-1)*req.PageSize), 0)

	dtos, total, err := h.service.GetPositions(ctx, req.UserId, limit, offset)
	if err != nil {
		slog.Error("gRPC GetPositions failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get positions: %v", err)
	}

	pbPositions := make([]*pb.Position, 0, len(dtos))
	for _, dto := range dtos {
		pbPositions = append(pbPositions, h.toProtoPosition(dto))
	}

	slog.Debug("gRPC GetPositions successful", "user_id", req.UserId, "count", len(pbPositions), "duration", time.Since(start))
	return &pb.GetPositionsResponse{
		Positions: pbPositions,
		Total:     total,
	}, nil
}

// GetPosition 获取持仓详情
// 处理 gRPC GetPosition 请求
func (h *GRPCHandler) GetPosition(ctx context.Context, req *pb.GetPositionRequest) (*pb.GetPositionResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetPosition received", "position_id", req.PositionId)

	dto, err := h.service.GetPosition(ctx, req.PositionId)
	if err != nil {
		slog.Error("gRPC GetPosition failed", "position_id", req.PositionId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get position: %v", err)
	}

	slog.Debug("gRPC GetPosition successful", "position_id", req.PositionId, "duration", time.Since(start))
	return &pb.GetPositionResponse{
		Position: h.toProtoPosition(dto),
	}, nil
}

// ClosePosition 平仓
// 处理 gRPC ClosePosition 请求
func (h *GRPCHandler) ClosePosition(ctx context.Context, req *pb.ClosePositionRequest) (*pb.ClosePositionResponse, error) {
	start := time.Now()
	slog.Info("gRPC ClosePosition received", "position_id", req.PositionId, "close_price", req.ClosePrice)

	closePrice, err := decimal.NewFromString(req.ClosePrice)
	if err != nil {
		slog.Warn("gRPC ClosePosition invalid price", "position_id", req.PositionId, "price", req.ClosePrice, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid close price: %v", err)
	}

	err = h.service.ClosePosition(ctx, req.PositionId, closePrice)
	if err != nil {
		slog.Error("gRPC ClosePosition failed", "position_id", req.PositionId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to close position: %v", err)
	}

	// 获取更新后的头寸以返回
	dto, err := h.service.GetPosition(ctx, req.PositionId)
	if err != nil {
		slog.Error("gRPC GetPosition after close failed", "position_id", req.PositionId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get updated position: %v", err)
	}

	slog.Info("gRPC ClosePosition successful", "position_id", req.PositionId, "duration", time.Since(start))
	return &pb.ClosePositionResponse{
		Position: h.toProtoPosition(dto),
	}, nil
}

// TccTryFreeze TCC Try: 预冻结持仓
func (h *GRPCHandler) TccTryFreeze(ctx context.Context, req *pb.TccPositionRequest) (*pb.TccPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid quantity: %v", err)
	}

	if err := h.service.TccTryFreeze(ctx, barrier, req.UserId, req.Symbol, quantity); err != nil {
		slog.Error("TccTryFreeze failed", "user_id", req.UserId, "symbol", req.Symbol, "error", err)
		return nil, status.Errorf(codes.Aborted, "TccTryFreeze failed: %v", err)
	}

	return &pb.TccPositionResponse{Success: true}, nil
}

// TccConfirmFreeze TCC Confirm: 确认冻结
func (h *GRPCHandler) TccConfirmFreeze(ctx context.Context, req *pb.TccPositionRequest) (*pb.TccPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	quantity, _ := decimal.NewFromString(req.Quantity)
	if err := h.service.TccConfirmFreeze(ctx, barrier, req.UserId, req.Symbol, quantity); err != nil {
		return nil, status.Errorf(codes.Internal, "TccConfirmFreeze failed: %v", err)
	}

	return &pb.TccPositionResponse{Success: true}, nil
}

// TccCancelFreeze TCC Cancel: 取消冻结
func (h *GRPCHandler) TccCancelFreeze(ctx context.Context, req *pb.TccPositionRequest) (*pb.TccPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	quantity, _ := decimal.NewFromString(req.Quantity)
	if err := h.service.TccCancelFreeze(ctx, barrier, req.UserId, req.Symbol, quantity); err != nil {
		return nil, status.Errorf(codes.Internal, "TccCancelFreeze failed: %v", err)
	}

	return &pb.TccPositionResponse{Success: true}, nil
}

// SagaDeductFrozen Saga 正向: 扣除冻结持仓
func (h *GRPCHandler) SagaDeductFrozen(ctx context.Context, req *pb.SagaPositionRequest) (*pb.SagaPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "barrier error: %v", err)
	}
	qty, _ := decimal.NewFromString(req.Quantity)
	if err := h.service.SagaDeductFrozen(ctx, barrier, req.UserId, req.Symbol, qty); err != nil {
		return nil, status.Errorf(codes.Aborted, "SagaDeductFrozen failed: %v", err)
	}
	return &pb.SagaPositionResponse{Success: true}, nil
}

// SagaRefundFrozen Saga 补偿: 恢复冻结持仓
func (h *GRPCHandler) SagaRefundFrozen(ctx context.Context, req *pb.SagaPositionRequest) (*pb.SagaPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "barrier error: %v", err)
	}
	qty, _ := decimal.NewFromString(req.Quantity)
	if err := h.service.SagaRefundFrozen(ctx, barrier, req.UserId, req.Symbol, qty); err != nil {
		return nil, status.Errorf(codes.Internal, "SagaRefundFrozen failed: %v", err)
	}
	return &pb.SagaPositionResponse{Success: true}, nil
}

// SagaAddPosition Saga 正向: 增加持仓
func (h *GRPCHandler) SagaAddPosition(ctx context.Context, req *pb.SagaPositionRequest) (*pb.SagaPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "barrier error: %v", err)
	}
	qty, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)
	if err := h.service.SagaAddPosition(ctx, barrier, req.UserId, req.Symbol, qty, price); err != nil {
		return nil, status.Errorf(codes.Aborted, "SagaAddPosition failed: %v", err)
	}
	return &pb.SagaPositionResponse{Success: true}, nil
}

// SagaSubPosition Saga 补偿: 扣除持仓
func (h *GRPCHandler) SagaSubPosition(ctx context.Context, req *pb.SagaPositionRequest) (*pb.SagaPositionResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "barrier error: %v", err)
	}
	qty, _ := decimal.NewFromString(req.Quantity)
	if err := h.service.SagaSubPosition(ctx, barrier, req.UserId, req.Symbol, qty); err != nil {
		return nil, status.Errorf(codes.Internal, "SagaSubPosition failed: %v", err)
	}
	return &pb.SagaPositionResponse{Success: true}, nil
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
