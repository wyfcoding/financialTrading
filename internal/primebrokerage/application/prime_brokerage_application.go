package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/primebrokerage/domain"
)

// RouteToSeatCommand 路由席位命令
type RouteToSeatCommand struct {
	Symbol   string
	Amount   int64
	Exchange string
}

// BorrowSecurityCommand 借券命令
type BorrowSecurityCommand struct {
	UserID   uint64
	Symbol   string
	Quantity int64
}

// ReturnSecurityCommand 还券命令
type ReturnSecurityCommand struct {
	LoanID string
}

// PrimeBrokerageApplicationService 主经纪商应用服务
type PrimeBrokerageApplicationService struct {
	repo        domain.PrimeBrokerageRepository
	router      domain.SeatRoutingStrategy
	poolService *domain.SecurityPoolService
	logger      *slog.Logger
}

func NewPrimeBrokerageApplicationService(
	repo domain.PrimeBrokerageRepository,
	router domain.SeatRoutingStrategy,
	poolService *domain.SecurityPoolService,
	logger *slog.Logger,
) *PrimeBrokerageApplicationService {
	return &PrimeBrokerageApplicationService{
		repo:        repo,
		router:      router,
		poolService: poolService,
		logger:      logger,
	}
}

// RouteToSeat 处理席位路由请求
func (s *PrimeBrokerageApplicationService) RouteToSeat(ctx context.Context, cmd RouteToSeatCommand) (*domain.ClearingSeat, error) {
	start := time.Now()
	seat, err := s.router.SelectSeat(ctx, cmd.Exchange, cmd.Amount)
	if err != nil {
		s.logger.Error("failed to route to seat", "symbol", cmd.Symbol, "exchange", cmd.Exchange, "error", err)
		return nil, err
	}

	duration := time.Since(start)
	s.logger.Info("routed to seat successfully", "seat_id", seat.ID, "duration", duration.String())
	return seat, nil
}

// BorrowSecurity 处理借券请求
func (s *PrimeBrokerageApplicationService) BorrowSecurity(ctx context.Context, cmd BorrowSecurityCommand) (*domain.SecurityLoan, error) {
	s.logger.Info("processing borrow security request", "user_id", cmd.UserID, "symbol", cmd.Symbol, "quantity", cmd.Quantity)

	pool, err := s.repo.FindPoolBySymbol(ctx, cmd.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to find security pool: %w", err)
	}

	if err := pool.Allocate(cmd.Quantity); err != nil {
		return nil, err
	}

	loan := &domain.SecurityLoan{
		LoanID:   fmt.Sprintf("LOAN-%d", time.Now().UnixNano()),
		UserID:   cmd.UserID,
		Symbol:   cmd.Symbol,
		Quantity: cmd.Quantity,
		Rate:     pool.MinRate,
		LoanedAt: time.Now(),
		DueAt:    time.Now().AddDate(0, 0, 7), // 默认7天
		Status:   "ACTIVE",
	}

	if err := s.repo.SaveLoan(ctx, loan); err != nil {
		return nil, fmt.Errorf("failed to save security loan: %w", err)
	}

	if err := s.repo.UpdatePool(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to update security pool: %w", err)
	}

	s.logger.Info("borrow security success", "loan_id", loan.LoanID)
	return loan, nil
}

// ReturnSecurity 处理还券请求
func (s *PrimeBrokerageApplicationService) ReturnSecurity(ctx context.Context, cmd ReturnSecurityCommand) error {
	loan, err := s.repo.FindLoanByID(ctx, cmd.LoanID)
	if err != nil {
		return err
	}

	if loan.Status != "ACTIVE" {
		return fmt.Errorf("loan %s is not active", cmd.LoanID)
	}

	pool, err := s.repo.FindPoolBySymbol(ctx, loan.Symbol)
	if err != nil {
		return err
	}

	pool.Deallocate(loan.Quantity)
	loan.Status = "RETURNED"

	if err := s.repo.SaveLoan(ctx, loan); err != nil {
		return err
	}

	if err := s.repo.UpdatePool(ctx, pool); err != nil {
		return err
	}

	s.logger.Info("return security success", "loan_id", cmd.LoanID)
	return nil
}
