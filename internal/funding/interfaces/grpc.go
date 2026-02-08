package interfaces

import (
	"context"

	pb "github.com/wyfcoding/financialtrading/go-api/funding/v1"
	"github.com/wyfcoding/financialtrading/internal/funding/application"
)

type FundingHandler struct {
	pb.UnimplementedFundingServiceServer
	app *application.FundingService
}

func NewFundingHandler(app *application.FundingService) *FundingHandler {
	return &FundingHandler{app: app}
}

func (h *FundingHandler) GetFundingRate(ctx context.Context, req *pb.GetFundingRateRequest) (*pb.GetFundingRateResponse, error) {
	return h.app.GetFundingRate(ctx, req.Symbol)
}

func (h *FundingHandler) RequestMarginLoan(ctx context.Context, req *pb.RequestMarginLoanRequest) (*pb.RequestMarginLoanResponse, error) {
	return h.app.RequestMarginLoan(ctx, req.UserId, req.Asset, req.Amount)
}

func (h *FundingHandler) RepayMarginLoan(ctx context.Context, req *pb.RepayMarginLoanRequest) (*pb.RepayMarginLoanResponse, error) {
	return h.app.RepayMarginLoan(ctx, req.LoanId, req.Amount)
}

func (h *FundingHandler) GetLoans(ctx context.Context, req *pb.GetLoansRequest) (*pb.GetLoansResponse, error) {
	return h.app.GetLoans(ctx, req.UserId)
}
