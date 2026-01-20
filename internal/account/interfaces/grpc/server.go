package grpc

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/account/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountGrpcServer struct {
	accountv1.UnimplementedAccountServiceServer
	appService   *application.AccountService
	queryService *application.AccountQueryService
}

func NewAccountGrpcServer(
	appService *application.AccountService,
	queryService *application.AccountQueryService,
) *AccountGrpcServer {
	return &AccountGrpcServer{
		appService:   appService,
		queryService: queryService,
	}
}

// CreateAccount 开户
func (s *AccountGrpcServer) CreateAccount(ctx context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	slog.InfoContext(ctx, "grpc_create_account", "user_id", req.UserId, "currency", req.Currency)

	cmd := application.CreateAccountCommand{
		UserID:      req.UserId,
		AccountType: req.AccountType,
		Currency:    req.Currency,
	}

	dto, err := s.appService.CreateAccount(ctx, cmd)
	if err != nil {
		slog.ErrorContext(ctx, "grpc_create_account_failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &accountv1.CreateAccountResponse{
		Account: s.toProto(dto),
	}, nil
}

// GetAccount 获取账户
func (s *AccountGrpcServer) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	var dto *application.AccountDTO
	var err error

	if req.AccountId != "" {
		dto, err = s.queryService.GetAccount(ctx, req.AccountId)
	} else if req.UserId != "" {
		// 这里 accountv1.GetAccountRequest 并未明确指定 Currency，所以通常是 GetBalance 用的多
		// 但为了兼容 proto 定义，这里假设 GetAccount 只用 ID 查，或者返回列表中的第一个(不推荐)
		// 暂不支持仅通过 UseID 查单个Account Without Currency
		return nil, status.Error(codes.Unimplemented, "query by user_id only not supported in GetAccount, use GetBalance")
	} else {
		return nil, status.Error(codes.InvalidArgument, "account_id required")
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &accountv1.GetAccountResponse{
		Account: s.toProto(dto),
	}, nil
}

// GetBalance 查询余额
func (s *AccountGrpcServer) GetBalance(ctx context.Context, req *accountv1.GetBalanceRequest) (*accountv1.GetBalanceResponse, error) {
	// 这是一个特殊的查询，proto 定义 req 里没有 currency，但 response 有
	// 这通常意味着返回所有账户余额，或者 GetBalanceRequest 此处定义有歧义
	// 假设我们需要返回该用户所有币种余额，但 response 只定义了单个 string balance
	// 这里我们假设 GetBalanceRequest 应该包含 AccountID，如果没有，则无法精确查询

	if req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id required")
	}

	dto, err := s.queryService.GetAccount(ctx, req.AccountId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &accountv1.GetBalanceResponse{
		AccountId:        dto.AccountID,
		Balance:          dto.Balance,
		AvailableBalance: dto.AvailableBalance,
		FrozenBalance:    dto.FrozenBalance,
		Timestamp:        dto.UpdatedAt,
	}, nil
}

// Deposit 充值
func (s *AccountGrpcServer) Deposit(ctx context.Context, req *accountv1.DepositRequest) (*accountv1.DepositResponse, error) {
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}

	cmd := application.DepositCommand{
		AccountID: req.AccountId,
		Amount:    amount,
	}

	if err := s.appService.Deposit(ctx, cmd); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &accountv1.DepositResponse{
		AccountId: req.AccountId,
		Status:    "SUCCESS",
		Amount:    req.Amount,
		Timestamp: 0, // Fill if needed
	}, nil
}

// SagaDeductFrozen Saga 扣款
func (s *AccountGrpcServer) SagaDeductFrozen(ctx context.Context, req *accountv1.SagaAccountRequest) (*accountv1.SagaAccountResponse, error) {
	// Barrier usually comes from metadata or request. Here assuming none for demo.
	err := s.appService.SagaDeductFrozen(ctx, nil, req.UserId, req.Currency, req.Amount)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &accountv1.SagaAccountResponse{Success: true}, nil
}

// Other methods unimplemented for brevity in this step, but skeleton exists.

func (s *AccountGrpcServer) toProto(dto *application.AccountDTO) *accountv1.AccountResponse {
	if dto == nil {
		return nil
	}
	return &accountv1.AccountResponse{
		AccountId:        dto.AccountID,
		UserId:           dto.UserID,
		AccountType:      dto.AccountType,
		Currency:         dto.Currency,
		Balance:          dto.Balance,
		AvailableBalance: dto.AvailableBalance,
		CreatedAt:        0, // DTO needs field
		UpdatedAt:        dto.UpdatedAt,
	}
}
