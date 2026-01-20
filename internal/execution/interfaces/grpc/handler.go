// 包  gRPC 处理器实现
package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler 实现了 ExecutionService 的 gRPC 服务端接口，负责实时订单执行与复杂算法单的接收。
type Handler struct {
	pb.UnimplementedExecutionServiceServer
	service *application.ExecutionService // 关联的执行应用服务
}

// NewHandler 构造一个新的执行 gRPC 处理器实例。
func NewHandler(service *application.ExecutionService) *Handler {
	return &Handler{
		service: service,
	}
}

// ExecuteOrder 处理实时普通限价或市价单的执行请求。
func (h *Handler) ExecuteOrder(ctx context.Context, req *pb.ExecuteOrderRequest) (*pb.ExecuteOrderResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc execute_order received", "order_id", req.OrderId, "user_id", req.UserId, "symbol", req.Symbol)

	dto, err := h.service.ExecuteOrder(ctx, &application.ExecuteOrderRequest{
		OrderID:  req.OrderId,
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    req.Price,
		Quantity: req.Quantity,
	})
	if err != nil {
		slog.ErrorContext(ctx, "grpc execute_order failed", "order_id", req.OrderId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to execute order: %v", err)
	}

	slog.InfoContext(ctx, "grpc execute_order successful", "order_id", req.OrderId, "execution_id", dto.ExecutionID, "duration", time.Since(start))
	return &pb.ExecuteOrderResponse{
		ExecutionId:      dto.ExecutionID,
		OrderId:          dto.OrderID,
		Status:           dto.Status,
		ExecutedQuantity: dto.ExecutedQuantity,
		ExecutedPrice:    dto.ExecutedPrice,
		Timestamp:        dto.CreatedAt,
	}, nil
}

// GetExecutionHistory 分页查询用户的历史执行记录。
func (h *Handler) GetExecutionHistory(ctx context.Context, req *pb.GetExecutionHistoryRequest) (*pb.GetExecutionHistoryResponse, error) {
	start := time.Now()
	slog.DebugContext(ctx, "grpc get_execution_history received", "user_id", req.UserId)

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	dtos, _, err := h.service.GetExecutionHistory(ctx, req.UserId, limit, 0)
	if err != nil {
		slog.ErrorContext(ctx, "grpc get_execution_history failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
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

	slog.DebugContext(ctx, "grpc get_execution_history successful", "user_id", req.UserId, "duration", time.Since(start))
	return &pb.GetExecutionHistoryResponse{
		Executions: records,
	}, nil
}

// SubmitAlgoOrder 提交高级算法订单（如 VWAP）。
func (h *Handler) SubmitAlgoOrder(ctx context.Context, req *pb.SubmitAlgoOrderRequest) (*pb.SubmitAlgoOrderResponse, error) {
	if req.UserId == "" || req.Symbol == "" || req.TotalQuantity == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	resp, err := h.service.SubmitAlgoOrder(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "grpc submit_algo_order failed", "user_id", req.UserId, "symbol", req.Symbol, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to submit algo order: %v", err)
	}

	return resp, nil
}

// SubmitSOROrder 提交智能路由订单。
func (h *Handler) SubmitSOROrder(ctx context.Context, req *pb.SubmitSOROrderRequest) (*pb.SubmitSOROrderResponse, error) {
	if req.UserId == "" || req.Symbol == "" || req.TotalQuantity == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required fields")
	}

	resp, err := h.service.SubmitSOROrder(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "grpc submit_sor_order failed", "user_id", req.UserId, "symbol", req.Symbol, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to submit SOR order: %v", err)
	}

	return resp, nil
}
