package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/funding/v1"
	"github.com/wyfcoding/financialtrading/internal/funding/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FundingService struct {
	repo domain.FundingRepository
}

func NewFundingService(repo domain.FundingRepository) *FundingService {
	return &FundingService{repo: repo}
}

func (s *FundingService) GetFundingRate(ctx context.Context, symbol string) (*pb.GetFundingRateResponse, error) {
	rate, err := s.repo.GetLatestRate(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, fmt.Errorf("funding rate not found for %s", symbol)
	}

	return &pb.GetFundingRateResponse{
		Symbol:        rate.Symbol,
		FundingRate:   rate.Rate.String(),
		NextFundingAt: timestamppb.New(rate.Timestamp.Add(8 * time.Hour)), // 假设 8 小时一次
	}, nil
}

func (s *FundingService) RequestMarginLoan(ctx context.Context, userID, asset, amountStr string) (*pb.RequestMarginLoanResponse, error) {
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, err
	}

	// 模拟获取当前利率
	rate := decimal.NewFromFloat(0.0005) // 0.05% 日利息
	loanID := fmt.Sprintf("loan_%d", time.Now().UnixNano())

	loan := domain.NewMarginLoan(loanID, userID, asset, amount, rate)
	if err := s.repo.SaveLoan(ctx, loan); err != nil {
		return nil, err
	}

	return &pb.RequestMarginLoanResponse{
		LoanId: loan.LoanID,
		Status: "ACTIVE",
	}, nil
}

func (s *FundingService) RepayMarginLoan(ctx context.Context, loanID, amountStr string) (*pb.RepayMarginLoanResponse, error) {
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, err
	}

	loan, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		return nil, err
	}
	if loan == nil {
		return nil, fmt.Errorf("loan %s not found", loanID)
	}

	if err := loan.Repay(amount); err != nil {
		return nil, err
	}

	if err := s.repo.SaveLoan(ctx, loan); err != nil {
		return nil, err
	}

	return &pb.RepayMarginLoanResponse{Success: true}, nil
}

func (s *FundingService) GetLoans(ctx context.Context, userID string) (*pb.GetLoansResponse, error) {
	loans, err := s.repo.ListLoans(ctx, userID)
	if err != nil {
		return nil, err
	}

	var pbLoans []*pb.LoanItem
	for _, l := range loans {
		pbLoans = append(pbLoans, &pb.LoanItem{
			LoanId:    l.LoanID,
			Asset:     l.Asset,
			Principal: l.Principal.String(),
			Interest:  l.Interest.String(),
			Status:    fmt.Sprintf("%d", l.Status),
			CreatedAt: timestamppb.New(l.CreatedAt),
		})
	}

	return &pb.GetLoansResponse{Loans: pbLoans}, nil
}
