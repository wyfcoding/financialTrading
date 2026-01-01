// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/goapi/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与订单执行相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedExecutionServiceServer
	service *application.ExecutionService // 执行应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// service: 注入的执行应用服务
func NewGRPCHandler(service *application.ExecutionService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// ExecuteOrder 执行订单
// 处理 gRPC ExecuteOrder 请求
// 接收 Proto 定义的订单参数，调用应用服务执行订单，并返回 Proto 定义的响应
func (h *GRPCHandler) ExecuteOrder(ctx context.Context, req *pb.ExecuteOrderRequest) (*pb.ExecuteOrderResponse, error) {
	start := time.Now()
	slog.Info("gRPC ExecuteOrder received", "order_id", req.OrderId, "user_id", req.UserId, "symbol", req.Symbol, "side", req.Side)

	// 调用应用服务执行订单
	dto, err := h.service.ExecuteOrder(ctx, &application.ExecuteOrderRequest{
		OrderID:  req.OrderId,
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    req.Price,
		Quantity: req.Quantity,
	})
	if err != nil {
		slog.Error("gRPC ExecuteOrder failed", "order_id", req.OrderId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to execute order: %v", err)
	}

	slog.Info("gRPC ExecuteOrder successful", "order_id", req.OrderId, "execution_id", dto.ExecutionID, "status", dto.Status, "duration", time.Since(start))
	// 返回执行结果
	return &pb.ExecuteOrderResponse{
		ExecutionId:      dto.ExecutionID,
		OrderId:          dto.OrderID,
		Status:           dto.Status,
		ExecutedQuantity: dto.ExecutedQuantity,
		ExecutedPrice:    dto.ExecutedPrice,
		Timestamp:        dto.CreatedAt,
	}, nil
}

// GetExecutionHistory 获取执行历史
// 处理 gRPC GetExecutionHistory 请求
func (h *GRPCHandler) GetExecutionHistory(ctx context.Context, req *pb.GetExecutionHistoryRequest) (*pb.GetExecutionHistoryResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetExecutionHistory received", "user_id", req.UserId, "limit", req.Limit)

	// 如果未指定限制，则使用默认值
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	dtos, _, err := h.service.GetExecutionHistory(ctx, req.UserId, limit, 0)
	if err != nil {
		slog.Error("gRPC GetExecutionHistory failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get execution history: %v", err)
	}

	records := make([]*pb.ExecutionRecord, 0, len(dtos))
	for _, dto := range dtos {
		records = append(records, &pb.ExecutionRecord{
			ExecutionId:      dto.ExecutionID,
			OrderId:          dto.OrderID,
			Symbol:           dto.Symbol,
			ExecutedQuantity: dto.ExecutedQuantity,
			ExecutedPrice:    dto.ExecutedPrice,
			Timestamp:        dto.CreatedAt,
		})
	}

	slog.Debug("gRPC GetExecutionHistory successful", "user_id", req.UserId, "records_count", len(records), "duration", time.Since(start))
	return &pb.GetExecutionHistoryResponse{
		Executions: records,
	}, nil
}

// SubmitAlgoOrder 提交算法订单。
func (h *GRPCHandler) SubmitAlgoOrder(ctx context.Context, req *pb.SubmitAlgoOrderRequest) (*pb.SubmitAlgoOrderResponse, error) {
	if req.UserId == "" || req.Symbol == "" || req.TotalQuantity == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	resp, err := h.service.SubmitAlgoOrder(ctx, req)
	if err != nil {
		slog.Error("gRPC SubmitAlgoOrder failed", "user_id", req.UserId, "symbol", req.Symbol, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to submit algo order: %v", err)
	}

	return resp, nil
}

// SubmitSOROrder 提交智能路由订单。
func (h *GRPCHandler) SubmitSOROrder(ctx context.Context, req *pb.SubmitSOROrderRequest) (*pb.SubmitSOROrderResponse, error) {
	if req.UserId == "" || req.Symbol == "" || req.TotalQuantity == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	resp, err := h.service.SubmitSOROrder(ctx, req)
	if err != nil {
		slog.Error("gRPC SubmitSOROrder failed", "user_id", req.UserId, "symbol", req.Symbol, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to submit SOR order: %v", err)
	}

	return resp, nil
}
