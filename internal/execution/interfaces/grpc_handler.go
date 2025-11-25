package interfaces

import (
	"context"

	pb "github.com/fynnwu/FinancialTrading/go-api/execution"
	"github.com/fynnwu/FinancialTrading/internal/execution/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedExecutionServiceServer
	appService *application.ExecutionApplicationService
}

func NewGRPCHandler(appService *application.ExecutionApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

func (h *GRPCHandler) ExecuteOrder(ctx context.Context, req *pb.ExecuteOrderRequest) (*pb.ExecutionResponse, error) {
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

	return &pb.ExecutionResponse{
		ExecutionId:      dto.ExecutionID,
		OrderId:          dto.OrderID,
		Status:           dto.Status,
		ExecutedQuantity: dto.ExecutedQuantity,
		ExecutedPrice:    dto.ExecutedPrice,
		Timestamp:        dto.CreatedAt,
	}, nil
}

func (h *GRPCHandler) GetExecutionHistory(ctx context.Context, req *pb.GetExecutionHistoryRequest) (*pb.ExecutionHistoryResponse, error) {
	// Default limit if not specified
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

	return &pb.ExecutionHistoryResponse{
		Executions: records,
	}, nil
}
