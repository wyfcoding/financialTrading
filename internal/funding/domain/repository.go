package domain

import (
	"context"
)

type FundingRepository interface {
	SaveLoan(ctx context.Context, loan *MarginLoan) error
	GetLoan(ctx context.Context, loanID string) (*MarginLoan, error)
	ListLoans(ctx context.Context, userID string) ([]*MarginLoan, error)

	SaveRate(ctx context.Context, rate *FundingRate) error
	GetLatestRate(ctx context.Context, symbol string) (*FundingRate, error)
}
