package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialTrading/go-api/feemanagement/v1"
	"github.com/wyfcoding/financialTrading/internal/feemanagement/domain"
	"github.com/wyfcoding/pkg/idgen"
)

type FeeService struct {
	repo   domain.FeeRepository
	idGen  idgen.Generator
	logger *slog.Logger
}

func NewFeeService(repo domain.FeeRepository, idGen idgen.Generator, logger *slog.Logger) *FeeService {
	return &FeeService{
		repo:   repo,
		idGen:  idGen,
		logger: logger.With("service", "feemanagement_application"),
	}
}

func (s *FeeService) CreateSchedule(ctx context.Context, req *pb.CreateFeeScheduleRequest) (*domain.FeeSchedule, error) {
	schedule := &domain.FeeSchedule{
		ID:         fmt.Sprintf("fee_%d", s.idGen.Generate()),
		Name:       req.Name,
		UserTier:   req.UserTier,
		AssetClass: req.AssetClass,
		BaseRate:   req.BaseRate,
		MinFee:     req.MinFee,
		MaxFee:     req.MaxFee,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.SaveSchedule(ctx, schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

func (s *FeeService) EstimateFees(ctx context.Context, userID, symbol, assetClass string, amount float64) (*domain.TradeFeeRecord, error) {
	// TODO: 从 user 服务获取用户的 Tier
	userTier := "standard"

	schedule, err := s.repo.GetScheduleByTier(ctx, userTier, assetClass)
	if err != nil {
		// 返回默认费率或错误
		return nil, fmt.Errorf("fee schedule not found for tier %s: %w", userTier, err)
	}

	total := schedule.Calculate(amount)

	return &domain.TradeFeeRecord{
		UserID:   userID,
		TotalFee: total,
		Currency: "USD",
		Components: []domain.FeeComponent{
			{Type: pb.FeeType_FEE_TYPE_COMMISSION, Amount: total, Currency: "USD", Description: "Commission based on base rate"},
		},
		CalculatedAt: time.Now(),
	}, nil
}

func (s *FeeService) CalculateTradeFees(ctx context.Context, tradeID, orderID, userID, symbol, assetClass string, amount float64) (*domain.TradeFeeRecord, error) {
	res, err := s.EstimateFees(ctx, userID, symbol, assetClass, amount)
	if err != nil {
		return nil, err
	}

	res.TradeID = tradeID
	res.OrderID = orderID

	if err := s.repo.SaveTradeFee(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *FeeService) ListSchedules(ctx context.Context) ([]*domain.FeeSchedule, error) {
	return s.repo.ListSchedules(ctx)
}
