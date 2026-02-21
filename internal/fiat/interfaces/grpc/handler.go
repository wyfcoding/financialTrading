package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/fiat/v1"
	"github.com/wyfcoding/financialtrading/internal/fiat/application"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FiatHandler struct {
	pb.UnimplementedFiatServiceServer
	appService *application.FiatApplicationService
}

func NewFiatHandler(appService *application.FiatApplicationService) *FiatHandler {
	return &FiatHandler{appService: appService}
}

func (h *FiatHandler) GetRate(ctx context.Context, req *pb.GetRateRequest) (*pb.GetRateResponse, error) {
	rate, err := h.appService.GetRate(ctx, req.FromCurrency, req.ToCurrency)
	if err != nil {
		return nil, err
	}
	return &pb.GetRateResponse{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Rate:         rate,
		UpdatedAt:    timestamppb.Now(),
	}, nil
}

func (h *FiatHandler) Exchange(ctx context.Context, req *pb.ExchangeRequest) (*pb.ExchangeResponse, error) {
	cmd := &application.ExchangeCommand{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Amount:       decimal.NewFromFloat(req.Amount),
	}
	result, err := h.appService.Exchange(ctx, cmd)
	if err != nil {
		return nil, err
	}

	fromAmt, _ := result.FromAmount.Float64()
	toAmt, _ := result.ToAmount.Float64()
	rate, _ := result.Rate.Float64()

	return &pb.ExchangeResponse{
		FromAmount:   fromAmt,
		ToAmount:     toAmt,
		Rate:         rate,
		FromCurrency: result.FromCurrency,
		ToCurrency:   result.ToCurrency,
	}, nil
}

func (h *FiatHandler) LockRate(ctx context.Context, req *pb.LockRateRequest) (*pb.LockRateResponse, error) {
	cmd := &application.LockRateCommand{
		UserID:       req.UserId,
		PaymentID:    req.PaymentId,
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Amount:       decimal.NewFromFloat(req.Amount),
	}
	lock, err := h.appService.LockRate(ctx, cmd)
	if err != nil {
		return nil, err
	}

	rate, _ := lock.LockedRate.Float64()
	amt, _ := lock.LockedAmount.Float64()

	return &pb.LockRateResponse{
		LockId:       lock.ID,
		LockedRate:   rate,
		LockedAmount: amt,
		ExpiresAt:    timestamppb.New(lock.ExpiresAt),
	}, nil
}

func (h *FiatHandler) VerifyLock(ctx context.Context, req *pb.VerifyLockRequest) (*pb.VerifyLockResponse, error) {
	lock, err := h.appService.VerifyLock(ctx, req.LockId)
	if err != nil {
		return &pb.VerifyLockResponse{Valid: false}, nil
	}

	rate, _ := lock.LockedRate.Float64()

	return &pb.VerifyLockResponse{
		Valid:      true,
		UserId:     lock.UserID,
		PaymentId:  lock.PaymentID,
		LockedRate: rate,
	}, nil
}
