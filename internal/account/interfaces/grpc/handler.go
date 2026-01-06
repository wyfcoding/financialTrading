// Package grpc 提供了账户服务的 gRPC 接口实现。
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/goapi/account/v1"
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler 实现了 AccountService 的 gRPC 服务端接口，负责处理资金账户相关的远程调用。
type GRPCHandler struct {
	pb.UnimplementedAccountServiceServer
	service *application.AccountService // 关联的账户应用服务
}

// NewGRPCHandler 构造一个新的账户 gRPC 处理器实例。
func NewGRPCHandler(service *application.AccountService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// CreateAccount 处理创建新账户的请求。
func (h *GRPCHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc create_account received", "user_id", req.UserId, "currency", req.Currency)

	dto, err := h.service.CreateAccount(ctx, &application.CreateAccountRequest{
		UserID:      req.UserId,
		AccountType: req.AccountType,
		Currency:    req.Currency,
	})
	if err != nil {
		slog.ErrorContext(ctx, "grpc create_account failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	slog.InfoContext(ctx, "grpc create_account successful", "user_id", req.UserId, "account_id", dto.AccountID, "duration", time.Since(start))
	return &pb.CreateAccountResponse{
		Account: &pb.AccountResponse{
			AccountId:        dto.AccountID,
			UserId:           dto.UserID,
			AccountType:      dto.AccountType,
			Currency:         dto.Currency,
			Balance:          dto.Balance,
			AvailableBalance: dto.AvailableBalance,
			CreatedAt:        dto.CreatedAt,
			UpdatedAt:        dto.UpdatedAt,
		},
	}, nil
}

// GetAccount 获取指定账户的详细信息。
func (h *GRPCHandler) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	start := time.Now()
	slog.DebugContext(ctx, "grpc get_account received", "account_id", req.AccountId)

	dto, err := h.service.GetAccount(ctx, req.AccountId)
	if err != nil {
		slog.ErrorContext(ctx, "grpc get_account failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	slog.DebugContext(ctx, "grpc get_account successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.GetAccountResponse{
		Account: &pb.AccountResponse{
			AccountId:        dto.AccountID,
			UserId:           dto.UserID,
			AccountType:      dto.AccountType,
			Currency:         dto.Currency,
			Balance:          dto.Balance,
			AvailableBalance: dto.AvailableBalance,
			CreatedAt:        dto.CreatedAt,
			UpdatedAt:        dto.UpdatedAt,
		},
	}, nil
}

// Deposit 执行账户资金充值操作。
func (h *GRPCHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc deposit received", "account_id", req.AccountId, "amount", req.Amount)

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		slog.WarnContext(ctx, "grpc deposit invalid amount format", "account_id", req.AccountId, "amount", req.Amount, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	err = h.service.Deposit(ctx, req.AccountId, amount)
	if err != nil {
		slog.ErrorContext(ctx, "grpc deposit failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to deposit: %v", err)
	}

	slog.InfoContext(ctx, "grpc deposit successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.DepositResponse{
		AccountId: req.AccountId,
		Type:      "DEPOSIT",
		Amount:    req.Amount,
		Status:    "COMPLETED",
	}, nil
}

// GetBalance 获取指定账户的余额详情快照。
func (h *GRPCHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	start := time.Now()
	slog.DebugContext(ctx, "grpc get_balance received", "account_id", req.AccountId)

	dto, err := h.service.GetAccount(ctx, req.AccountId)
	if err != nil {
		slog.ErrorContext(ctx, "grpc get_balance failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get balance: %v", err)
	}

	slog.DebugContext(ctx, "grpc get_balance successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.GetBalanceResponse{
		AccountId:        dto.AccountID,
		Balance:          dto.Balance,
		AvailableBalance: dto.AvailableBalance,
		FrozenBalance:    dto.FrozenBalance,
		Timestamp:        dto.UpdatedAt,
	}, nil
}

// SagaDeductFrozen 执行 Saga 正向阶段：从冻结余额中真实扣除款项。
func (h *GRPCHandler) SagaDeductFrozen(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.service.SagaDeductFrozen(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_deduct_frozen execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "SagaDeductFrozen failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// SagaRefundFrozen 执行 Saga 补偿阶段：将之前扣除的资金退回。
func (h *GRPCHandler) SagaRefundFrozen(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.service.SagaRefundFrozen(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_refund_frozen execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "SagaRefundFrozen failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// SagaAddBalance 执行 Saga 正向阶段：直接增加用户的可用余额。
func (h *GRPCHandler) SagaAddBalance(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.service.SagaAddBalance(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_add_balance execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "SagaAddBalance failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// SagaSubBalance 执行 Saga 补偿阶段：扣除之前通过正向操作增加的余额。
func (h *GRPCHandler) SagaSubBalance(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.service.SagaSubBalance(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_sub_balance execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "SagaSubBalance failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// TccTryFreeze 执行 TCC 模式第一阶段：尝试预冻结资金。
func (h *GRPCHandler) TccTryFreeze(ctx context.Context, req *pb.TccFreezeRequest) (*pb.TccFreezeResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	if err := h.service.TccTryFreeze(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "tcc_try_freeze execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "TccTryFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

// TccConfirmFreeze 执行 TCC 模式第二阶段：确认冻结完成。
func (h *GRPCHandler) TccConfirmFreeze(ctx context.Context, req *pb.TccFreezeRequest) (*pb.TccFreezeResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	if err := h.service.TccConfirmFreeze(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "tcc_confirm_freeze execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "TccConfirmFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

// TccCancelFreeze 执行 TCC 模式第三阶段：取消并释放冻结资金。
func (h *GRPCHandler) TccCancelFreeze(ctx context.Context, req *pb.TccFreezeRequest) (*pb.TccFreezeResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	if err := h.service.TccCancelFreeze(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "tcc_cancel_freeze execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "TccCancelFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}