// 包  gRPC 处理器实现
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/goapi/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与订单执行相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedExecutionServiceServer
	appService *application.ExecutionApplicationService // 执行应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的执行应用服务
func NewGRPCHandler(appService *application.ExecutionApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// ExecuteOrder 执行订单
// 处理 gRPC ExecuteOrder 请求
// 接收 Proto 定义的订单参数，调用应用服务执行订单，并返回 Proto 定义的响应
func (h *GRPCHandler) ExecuteOrder(ctx context.Context, req *pb.ExecuteOrderRequest) (*pb.ExecuteOrderResponse, error) {
	// 调用应用服务执行订单
	dto, err := h.appService.ExecuteOrder(ctx, &application.ExecuteOrderRequest{
		OrderID:  req.OrderId,
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    req.Price,
		Quantity: req.Quantity,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to execute order: %v", err)
	}

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
	// 如果未指定限制，则使用默认值
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	dtos, _, err := h.appService.GetExecutionHistory(ctx, req.UserId, limit, 0)
	if err != nil {
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

	return &pb.GetExecutionHistoryResponse{
		Executions: records,
	}, nil
}
