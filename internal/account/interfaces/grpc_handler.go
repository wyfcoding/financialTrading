package interfaces

import (
	"context"

	pb "github.com/fynnwu/FinancialTrading/go-api/account"
	"github.com/fynnwu/FinancialTrading/internal/account/application"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedAccountServiceServer
	appService *application.AccountApplicationService
}

func NewGRPCHandler(appService *application.AccountApplicationService) *GRPCHandler {
	return &GRPCHandler{
		appService: appService,
	}
}

func (h *GRPCHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.AccountResponse, error) {
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
