package application

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialTrading/internal/marginlending/domain"
)

// EvaluateMarginCommand 保证金评估命令
type EvaluateMarginCommand struct {
	UserID   uint64
	Symbol   string
	Quantity int64
	Price    int64
}

// LockCollateralCommand 抵押锁定命令
type LockCollateralCommand struct {
	UserID uint64
	Asset  string
	Amount int64
}

// MarginLendingApplicationService 融资融券应用服务
type MarginLendingApplicationService struct {
	marginService *domain.MarginService
	repo          domain.MarginRepository
	logger        *slog.Logger
}

func NewMarginLendingApplicationService(marginService *domain.MarginService, repo domain.MarginRepository, logger *slog.Logger) *MarginLendingApplicationService {
	return &MarginLendingApplicationService{
		marginService: marginService,
		repo:          repo,
		logger:        logger,
	}
}

func (s *MarginLendingApplicationService) EvaluateMargin(ctx context.Context, cmd EvaluateMarginCommand) (*domain.MarginRequirement, error) {
	s.logger.Info("evaluating margin", "user_id", cmd.UserID, "symbol", cmd.Symbol)
	// 此处逻辑调用 domain 层的算法
	req := s.marginService.CalculateRequirement(cmd.Symbol, cmd.Quantity, cmd.Price)
	return req, nil
}

func (s *MarginLendingApplicationService) LockCollateral(ctx context.Context, cmd LockCollateralCommand) (string, error) {
	s.logger.Info("locking collateral", "user_id", cmd.UserID, "asset", cmd.Asset, "amount", cmd.Amount)
	// 基础设施层交互...
	return "LOCK-123", nil
}
