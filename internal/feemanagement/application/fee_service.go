package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/wyfcoding/financialtrading/go-api/feemanagement/v1"
	"github.com/wyfcoding/financialtrading/internal/feemanagement/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// FeeService 提供手续费与返佣相关用例
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

// CreateSchedule 创建手续费率表
func (s *FeeService) CreateSchedule(ctx context.Context, req *pb.CreateFeeScheduleRequest) (*domain.FeeSchedule, error) {
	schedule := &domain.FeeSchedule{
		ID:         fmt.Sprintf("fee_%d", s.idGen.Generate()),
		Name:       req.Name,
		UserTier:   req.UserTier,
		AssetClass: req.AssetClass,
		BaseRate:   req.BaseRate,
		MinFee:     req.MinFee,
		MaxFee:     req.MaxFee,
	}

	if err := s.repo.SaveSchedule(ctx, schedule); err != nil {
		s.logger.Error("failed to create fee schedule", "error", err)
		return nil, err
	}
	return schedule, nil
}

// EstimateFees 提供盘前预估手续费能力
func (s *FeeService) EstimateFees(ctx context.Context, userID, symbol, assetClass string, amount float64) (*domain.TradeFeeRecord, error) {
	// TODO: 可以通过 gRPC 从 UserProfile / Auth 服务获取用户对应的 tier
	// 这里通过简单的 stub 模拟。
	userTier := "standard"

	schedule, err := s.repo.GetScheduleByTier(ctx, userTier, assetClass)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee schedule for tier %s: %w", userTier, err)
	}

	total := schedule.Calculate(amount)

	return &domain.TradeFeeRecord{
		UserID:   userID,
		TotalFee: total,
		Currency: "USD",
		Components: []domain.FeeComponent{
			{
				Type:        int32(pb.FeeType_FEE_TYPE_COMMISSION),
				Amount:      total,
				Currency:    "USD",
				Description: "Commission based on base rate",
			},
		},
		CalculatedAt: time.Now(),
	}, nil
}

// CalculateTradeFees 对已成交交易进行真实的手续费核算与落库
func (s *FeeService) CalculateTradeFees(ctx context.Context, tradeID, orderID, userID, symbol, assetClass string, amount float64) (*domain.TradeFeeRecord, error) {
	// 1. 获取预估的费率
	res, err := s.EstimateFees(ctx, userID, symbol, assetClass, amount)
	if err != nil {
		return nil, err
	}

	// 2. 追加交易及订单信息
	res.TradeID = tradeID
	res.OrderID = orderID

	for i := range res.Components {
		res.Components[i].TradeFeeRecordID = res.TradeID
	}

	// 3. 落库记录
	if err := s.repo.SaveTradeFee(ctx, res); err != nil {
		s.logger.Error("failed to save trade fee record", "error", err, "trade_id", tradeID)
		return nil, err
	}
	return res, nil
}

// GetFeeSchedule 获取单个手续费率表
func (s *FeeService) GetFeeSchedule(ctx context.Context, id string) (*domain.FeeSchedule, error) {
	return s.repo.GetSchedule(ctx, id)
}

// ListSchedules 列出符合条件的手续费率表
func (s *FeeService) ListSchedules(ctx context.Context, tier, assetClass string) ([]*domain.FeeSchedule, error) {
	return s.repo.ListSchedules(ctx, tier, assetClass)
}

// CalculateRebate 计算代理商/用户的历史返佣
func (s *FeeService) CalculateRebate(ctx context.Context, userID string, startTime, endTime time.Time) (float64, error) {
	return s.repo.CalculateRebate(ctx, userID, startTime, endTime)
}
