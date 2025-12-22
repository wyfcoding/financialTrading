// 包  gRPC 处理器（Handler）的实现。
// 这一层是接口层（Interfaces Layer）的一部分，负责适配外部的 gRPC 请求，
// 并将其转换为对应用层（Application Layer）的调用。
package grpc

import (
	"context"

	pb "github.com/wyfcoding/financialTrading/go-api/clearing"
	"github.com/wyfcoding/financialTrading/internal/clearing/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler 是清算服务的 gRPC 处理器。
// 它实现了由 apoch 生成的 `ClearingServiceServer` 接口。
// 其核心职责是作为 gRPC 协议与内部应用逻辑之间的桥梁。
type GRPCHandler struct {
	pb.UnimplementedClearingServiceServer                                         // 嵌入未实现的 apoch 服务，确保向前兼容
	appService                            *application.ClearingApplicationService // 依赖注入的应用服务实例
}

// 创建 gRPC 处理器实例。
//
// @param appService 注入的清算应用服务实例。
// @return *GRPCHandler 返回一个新的 gRPC 处理器实例。
func NewGRPCHandler(appService *application.ClearingApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// SettleTrade 实现了 gRPC 的 SettleTrade 方法。
// 它接收 gRPC 请求，将其转换为应用层的 DTO，然后调用应用服务来处理。
func (h *GRPCHandler) SettleTrade(ctx context.Context, req *pb.SettleTradeRequest) (*pb.SettlementResponse, error) {
	// 1. 将 gRPC 请求对象 (*pb.SettleTradeRequest) 转换为应用层 DTO (*application.SettleTradeRequest)。
	//    这是接口层的核心职责之一：数据转换。
	appReq := &application.SettleTradeRequest{
		TradeID:    req.TradeId,
		BuyUserID:  req.BuyUserId,
		SellUserID: req.SellUserId,
		Symbol:     req.Symbol,
		Quantity:   req.Quantity,
		Price:      req.Price,
	}

	// 2. 调用应用服务来执行核心业务逻辑,接收返回的 settlementID。
	//    gRPC handler 本身不包含业务逻辑。
	settlementID, err := h.appService.SettleTrade(ctx, appReq)
	if err != nil {
		// 3. 错误处理：如果应用层返回错误，将其转换为标准的 gRPC 错误。
		//    使用 status.Errorf 可以附加 gRPC 状态码。
		return nil, status.Errorf(codes.Internal, "failed to settle trade: %v", err)
	}

	// 4. 构建并返回 gRPC 响应,填充从应用服务返回的 settlementID。
	return &pb.SettlementResponse{
		TradeId:      req.TradeId,
		Status:       "COMPLETED", // 假设状态为已完成
		SettlementId: settlementID,
	}, nil
}

// ExecuteEODClearing 实现了 gRPC 的 ExecuteEODClearing 方法。
func (h *GRPCHandler) ExecuteEODClearing(ctx context.Context, req *pb.ExecuteEODClearingRequest) (*pb.EODClearingResponse, error) {
	// 调用应用服务启动日终清算流程,接收返回的 clearingID。
	clearingID, err := h.appService.ExecuteEODClearing(ctx, req.ClearingDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to execute EOD clearing: %v", err)
	}

	// 返回包含 clearingID 的响应,表示任务已开始。
	return &pb.EODClearingResponse{
		Status:     "PROCESSING", // 表示任务已开始处理
		ClearingId: clearingID,
	}, nil
}

// GetClearingStatus 实现了 gRPC 的 GetClearingStatus 方法。
func (h *GRPCHandler) GetClearingStatus(ctx context.Context, req *pb.GetClearingStatusRequest) (*pb.ClearingStatusResponse, error) {
	// 调用应用服务获取清算任务状态。
	clearing, err := h.appService.GetClearingStatus(ctx, req.ClearingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get clearing status: %v", err)
	}
	// 如果应用服务返回 nil（表示未找到），则返回 gRPC 的 NotFound 错误。
	if clearing == nil {
		return nil, status.Errorf(codes.NotFound, "clearing task with id '%s' not found", req.ClearingId)
	}

	// 计算完成百分比
	var completionPercentage float32
	if clearing.TotalTrades > 0 {
		completionPercentage = float32(clearing.TradesSettled) / float32(clearing.TotalTrades) * 100
	}

	// 将从应用层获取的领域对象转换为 gRPC 响应对象。
	return &pb.ClearingStatusResponse{
		ClearingId:         clearing.ClearingID,
		Status:             clearing.Status,
		TradesProcessed:    clearing.TradesSettled,
		TradesTotal:        clearing.TotalTrades,
		ProgressPercentage: int64(completionPercentage),
	}, nil
}
