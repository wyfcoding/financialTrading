package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	executionv1 "github.com/wyfcoding/financialtrading/go-api/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ExecutionGrpcServer struct {
	executionv1.UnimplementedExecutionServiceServer
	app   *application.ExecutionApplicationService
	query *application.ExecutionQueryService
}

func NewExecutionGrpcServer(app *application.ExecutionApplicationService, query *application.ExecutionQueryService) *ExecutionGrpcServer {
	return &ExecutionGrpcServer{app: app, query: query}
}

// ExecuteOrder 执行订单
func (s *ExecutionGrpcServer) ExecuteOrder(ctx context.Context, req *executionv1.ExecuteOrderRequest) (*executionv1.ExecuteOrderResponse, error) {
	price, _ := decimal.NewFromString(req.Price)
	qty, _ := decimal.NewFromString(req.Quantity)

	cmd := application.ExecuteOrderCommand{
		OrderID:  req.OrderId,
		UserID:   req.UserId,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Price:    price,
		Quantity: qty,
	}

	dto, err := s.app.ExecuteOrder(ctx, cmd)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &executionv1.ExecuteOrderResponse{
		ExecutionId:      dto.ExecutionID,
		OrderId:          dto.OrderID,
		Status:           dto.Status,
		ExecutedQuantity: dto.ExecutedQty,
		ExecutedPrice:    dto.ExecutedPx,
		Timestamp:        dto.Timestamp,
	}, nil
}

// SubmitAlgoOrder 提交算法订单
func (s *ExecutionGrpcServer) SubmitAlgoOrder(ctx context.Context, req *executionv1.SubmitAlgoOrderRequest) (*executionv1.SubmitAlgoOrderResponse, error) {
	totalQty, _ := decimal.NewFromString(req.TotalQuantity)

	cmd := application.SubmitAlgoCommand{
		UserID:    req.UserId,
		Symbol:    req.Symbol,
		Side:      req.Side,
		TotalQty:  totalQty,
		AlgoType:  req.AlgoType,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Params: map[string]string{
			"participation_rate": req.ParticipationRate,
		},
	}

	algoID, err := s.app.SubmitAlgoOrder(ctx, cmd)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &executionv1.SubmitAlgoOrderResponse{
		AlgoId: algoID,
		Status: "ACCEPTED",
	}, nil
}

// GetExecutionHistory 历史查询
func (s *ExecutionGrpcServer) GetExecutionHistory(ctx context.Context, req *executionv1.GetExecutionHistoryRequest) (*executionv1.GetExecutionHistoryResponse, error) {
	// 简单实现：使用 symbol 作为 orderID 占位符演示
	// 实际应根据 user/symbol 查
	dtos, err := s.query.GetExecutionHistory(ctx, req.Symbol)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	records := make([]*executionv1.ExecutionRecord, len(dtos))
	for i, d := range dtos {
		records[i] = &executionv1.ExecutionRecord{
			ExecutionId:      d.ExecutionID,
			OrderId:          d.OrderID,
			Symbol:           req.Symbol, // Simplified
			ExecutedQuantity: d.ExecutedQty,
			ExecutedPrice:    d.ExecutedPx,
			Timestamp:        d.Timestamp,
		}
	}
	return &executionv1.GetExecutionHistoryResponse{Executions: records}, nil
}
