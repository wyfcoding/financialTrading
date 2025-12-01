// Package grpc 包含 gRPC 处理器实现
package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialTrading/go-api/account"
	"github.com/wyfcoding/financialTrading/internal/account/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler gRPC 处理器
// 负责处理与账户相关的 gRPC 请求
type GRPCHandler struct {
	pb.UnimplementedAccountServiceServer
	appService *application.AccountApplicationService // 账户应用服务
}

// NewGRPCHandler 创建 gRPC 处理器实例
// appService: 注入的账户应用服务
func NewGRPCHandler(appService *application.AccountApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

// CreateAccount 创建账户
// 处理 gRPC CreateAccount 请求
func (h *GRPCHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.AccountResponse, error) {
	// 调用应用服务创建账户
	dto, err := h.appService.CreateAccount(ctx, &application.CreateAccountRequest{
		UserID:      req.UserId,
		AccountType: req.AccountType,
		Currency:    req.Currency,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
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
	}, nil
}

func (h *GRPCHandler) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.AccountResponse, error) {
	dto, err := h.appService.GetAccount(ctx, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
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
	}, nil
}

func (h *GRPCHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.TransactionResponse, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	err = h.appService.Deposit(ctx, req.AccountId, amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to deposit: %v", err)
	}

	return &pb.TransactionResponse{
		AccountId: req.AccountId,
		Type:      "DEPOSIT",
		Amount:    req.Amount,
		Status:    "COMPLETED",
		// TransactionID and Timestamp would ideally be returned by the service
	}, nil
}

func (h *GRPCHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.BalanceResponse, error) {
	dto, err := h.appService.GetAccount(ctx, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	return &pb.BalanceResponse{
		AccountId:        dto.AccountID,
		Balance:          dto.Balance,
		AvailableBalance: dto.AvailableBalance,
		FrozenBalance:    dto.FrozenBalance,
		Timestamp:        dto.UpdatedAt,
	}, nil
}
