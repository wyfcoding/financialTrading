// Package grpc 提供了账户服务的 gRPC 接口实现。
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler 实现了 AccountService 的 gRPC 服务端接口，负责处理资金账户相关的远程调用。
type Handler struct {
	pb.UnimplementedAccountServiceServer
	appService   *application.AccountService
	queryService *application.AccountQueryService
}

// NewHandler 构造一个新的账户 gRPC 处理器实例。
func NewHandler(appService *application.AccountService, queryService *application.AccountQueryService) *Handler {
	return &Handler{
		appService:   appService,
		queryService: queryService,
	}
}

// CreateAccount 处理创建新账户的请求。
func (h *Handler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc create_account received", "user_id", req.UserId, "currency", req.Currency)

	dto, err := h.appService.CreateAccount(ctx, application.CreateAccountCommand{
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
		Account: h.toProto(dto),
	}, nil
}

// GetAccount 获取指定账户的详细信息。
func (h *Handler) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	start := time.Now()
	slog.DebugContext(ctx, "grpc get_account received", "account_id", req.AccountId)

	dto, err := h.queryService.GetAccount(ctx, req.AccountId)
	if err != nil {
		slog.ErrorContext(ctx, "grpc get_account failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	slog.DebugContext(ctx, "grpc get_account successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.GetAccountResponse{
		Account: h.toProto(dto),
	}, nil
}

// Deposit 执行账户资金充值操作。
func (h *Handler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	start := time.Now()
	slog.InfoContext(ctx, "grpc deposit received", "account_id", req.AccountId, "amount", req.Amount)

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		slog.WarnContext(ctx, "grpc deposit invalid amount format", "account_id", req.AccountId, "amount", req.Amount, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	err = h.appService.Deposit(ctx, application.DepositCommand{
		AccountID: req.AccountId,
		Amount:    amount,
	})
	if err != nil {
		slog.ErrorContext(ctx, "grpc deposit failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to deposit: %v", err)
	}

	slog.InfoContext(ctx, "grpc deposit successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.DepositResponse{
		AccountId: req.AccountId,
		Status:    "SUCCESS",
		Amount:    req.Amount,
		Timestamp: 0,
	}, nil
}

// GetBalance 获取指定账户的余额详情快照。
func (h *Handler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	start := time.Now()
	slog.DebugContext(ctx, "grpc get_balance received", "account_id", req.AccountId)

	dto, err := h.queryService.GetAccount(ctx, req.AccountId)
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
func (h *Handler) SagaDeductFrozen(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.appService.SagaDeductFrozen(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_deduct_frozen execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "SagaDeductFrozen failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// SagaRefundFrozen 执行 Saga 补偿阶段：将之前扣除的资金退回。
func (h *Handler) SagaRefundFrozen(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.appService.SagaRefundFrozen(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_refund_frozen execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "SagaRefundFrozen failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// SagaAddBalance 执行 Saga 正向阶段：直接增加用户的可用余额。
func (h *Handler) SagaAddBalance(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.appService.SagaAddBalance(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_add_balance execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "SagaAddBalance failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// SagaSubBalance 执行 Saga 补偿阶段：扣除之前通过正向操作增加的余额。
func (h *Handler) SagaSubBalance(ctx context.Context, req *pb.SagaAccountRequest) (*pb.SagaAccountResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount format: %v", err)
	}
	if err := h.appService.SagaSubBalance(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "saga_sub_balance execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "SagaSubBalance failed: %v", err)
	}
	return &pb.SagaAccountResponse{Success: true}, nil
}

// TccTryFreeze 执行 TCC 模式第一阶段：尝试预冻结资金。
func (h *Handler) TccTryFreeze(ctx context.Context, req *pb.TccFreezeRequest) (*pb.TccFreezeResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	if err := h.appService.TccTryFreeze(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "tcc_try_freeze execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "TccTryFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

// TccConfirmFreeze 执行 TCC 模式第二阶段：确认冻结完成。
func (h *Handler) TccConfirmFreeze(ctx context.Context, req *pb.TccFreezeRequest) (*pb.TccFreezeResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	if err := h.appService.TccConfirmFreeze(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "tcc_confirm_freeze execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "TccConfirmFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

// TccCancelFreeze 执行 TCC 模式第三阶段：取消并释放冻结资金。
func (h *Handler) TccCancelFreeze(ctx context.Context, req *pb.TccFreezeRequest) (*pb.TccFreezeResponse, error) {
	barrier, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dtm barrier: %v", err)
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	if err := h.appService.TccCancelFreeze(ctx, barrier, req.UserId, req.Currency, amount); err != nil {
		slog.ErrorContext(ctx, "tcc_cancel_freeze execution failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "TccCancelFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

func (h *Handler) toProto(dto *application.AccountDTO) *pb.AccountResponse {
	if dto == nil {
		return nil
	}
	return &pb.AccountResponse{
		AccountId:        dto.AccountID,
		UserId:           dto.UserID,
		AccountType:      dto.AccountType,
		Currency:         dto.Currency,
		Balance:          dto.Balance,
		AvailableBalance: dto.AvailableBalance,
		CreatedAt:        dto.CreatedAt,
		UpdatedAt:        dto.UpdatedAt,
	}
}
