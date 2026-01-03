// 包  gRPC 处理器实现
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

// GRPCHandler gRPC 处理器
// 负责处理与账户相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedAccountServiceServer
	service *application.AccountService // 账户应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// service: 注入的账户应用服务
func NewGRPCHandler(service *application.AccountService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// CreateAccount 创建账户
// 处理 gRPC CreateAccount 请求
func (h *GRPCHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	start := time.Now()
	slog.Info("gRPC CreateAccount received", "user_id", req.UserId, "account_type", req.AccountType, "currency", req.Currency)

	// 调用应用服务创建账户
	dto, err := h.service.CreateAccount(ctx, &application.CreateAccountRequest{
		UserID:      req.UserId,
		AccountType: req.AccountType,
		Currency:    req.Currency,
	})
	if err != nil {
		slog.Error("gRPC CreateAccount failed", "user_id", req.UserId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	slog.Info("gRPC CreateAccount successful", "user_id", req.UserId, "account_id", dto.AccountID, "duration", time.Since(start))
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

// GetAccount 获取账户信息
// 处理 gRPC GetAccount 请求
func (h *GRPCHandler) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetAccount received", "account_id", req.AccountId)

	dto, err := h.service.GetAccount(ctx, req.AccountId)
	if err != nil {
		slog.Error("gRPC GetAccount failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	slog.Debug("gRPC GetAccount successful", "account_id", req.AccountId, "duration", time.Since(start))
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

// Deposit 账户充值
// 处理 gRPC Deposit 请求
func (h *GRPCHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	start := time.Now()
	slog.Info("gRPC Deposit received", "account_id", req.AccountId, "amount", req.Amount)

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		slog.Warn("gRPC Deposit invalid amount", "account_id", req.AccountId, "amount", req.Amount, "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	err = h.service.Deposit(ctx, req.AccountId, amount)
	if err != nil {
		slog.Error("gRPC Deposit failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to deposit: %v", err)
	}

	slog.Info("gRPC Deposit successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.DepositResponse{
		AccountId: req.AccountId,
		Type:      "DEPOSIT",
		Amount:    req.Amount,
		Status:    "COMPLETED",
	}, nil
}

// GetBalance 获取账户余额
// 处理 gRPC GetBalance 请求
func (h *GRPCHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	start := time.Now()
	slog.Debug("gRPC GetBalance received", "account_id", req.AccountId)

	dto, err := h.service.GetAccount(ctx, req.AccountId)
	if err != nil {
		slog.Error("gRPC GetBalance failed", "account_id", req.AccountId, "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	slog.Debug("gRPC GetBalance successful", "account_id", req.AccountId, "duration", time.Since(start))
	return &pb.GetBalanceResponse{
		AccountId:        dto.AccountID,
		Balance:          dto.Balance,
		AvailableBalance: dto.AvailableBalance,
		FrozenBalance:    dto.FrozenBalance,
		Timestamp:        dto.UpdatedAt,
	}, nil
}

// TccTryFreeze TCC Try: 预冻结
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
		slog.Error("TccTryFreeze failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Aborted, "TccTryFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

// TccConfirmFreeze TCC Confirm: 确认冻结
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
		slog.Error("TccConfirmFreeze failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "TccConfirmFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}

// TccCancelFreeze TCC Cancel: 取消冻结
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
		slog.Error("TccCancelFreeze failed", "user_id", req.UserId, "error", err)
		return nil, status.Errorf(codes.Internal, "TccCancelFreeze failed: %v", err)
	}

	return &pb.TccFreezeResponse{Success: true}, nil
}
