package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/position/domain"
	"github.com/wyfcoding/pkg/logging"
)

// PositionCommandService 处理所有持仓相关的写入操作（Commands）。
type PositionCommandService struct {
	repo   domain.PositionRepository
	logger *slog.Logger
}

// NewPositionCommandService 构造函数。
func NewPositionCommandService(repo domain.PositionRepository, logger *slog.Logger) *PositionCommandService {
	return &PositionCommandService{
		repo:   repo,
		logger: logger.With("module", "position_command"),
	}
}

// ClosePosition 彻底平仓指定的持仓。
func (s *PositionCommandService) ClosePosition(ctx context.Context, positionID string, closePrice decimal.Decimal) error {
	if positionID == "" || closePrice.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("invalid request parameters")
	}

	if err := s.repo.Close(ctx, positionID, closePrice); err != nil {
		logging.Error(ctx, "failed to close position", "position_id", positionID, "error", err)
		return err
	}

	logging.Info(ctx, "position closed successfully", "position_id", positionID, "close_price", closePrice.String())
	return nil
}

// --- TCC Distributed Transaction Support ---

func (s *PositionCommandService) TccTryFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		pos, err := s.repo.GetByUserSymbol(ctx, userID, symbol)
		if err != nil {
			return err
		}
		if pos == nil {
			return fmt.Errorf("no open position found for %s", symbol)
		}

		requestQty := quantity.InexactFloat64()
		if pos.Quantity < requestQty {
			return fmt.Errorf("insufficient position quantity")
		}

		pos.Quantity -= requestQty
		return s.repo.Update(ctx, pos)
	})
}

func (s *PositionCommandService) TccConfirmFreeze(ctx context.Context, barrier any, _, symbol string, quantity decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		return nil
	})
}

func (s *PositionCommandService) TccCancelFreeze(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		pos, err := s.repo.GetByUserSymbol(ctx, userID, symbol)
		if err != nil {
			return err
		}
		if pos != nil {
			pos.Quantity += quantity.InexactFloat64()
			return s.repo.Update(ctx, pos)
		}
		return fmt.Errorf("position not found to rollback freeze")
	})
}

// --- Saga Distributed Transaction Support ---

func (s *PositionCommandService) SagaDeductFrozen(ctx context.Context, barrier any, userID, symbol string, quantity, price decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		pos, err := s.repo.GetByUserSymbol(ctx, userID, symbol)
		if err != nil || pos == nil {
			return fmt.Errorf("position not found")
		}

		pos.UpdatePosition("sell", quantity.InexactFloat64(), price.InexactFloat64())
		return s.repo.Update(ctx, pos)
	})
}

func (s *PositionCommandService) SagaRefundFrozen(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.TccCancelFreeze(ctx, barrier, userID, symbol, quantity)
}

func (s *PositionCommandService) SagaAddPosition(ctx context.Context, barrier any, userID, symbol string, quantity, price decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		pos, err := s.repo.GetByUserSymbol(ctx, userID, symbol)
		if err != nil {
			return err
		}

		if pos == nil {
			pos = domain.NewPosition(userID, symbol)
			pos.UpdatePosition("buy", quantity.InexactFloat64(), price.InexactFloat64())
			return s.repo.Save(ctx, pos)
		}

		pos.UpdatePosition("buy", quantity.InexactFloat64(), price.InexactFloat64())
		return s.repo.Update(ctx, pos)
	})
}

func (s *PositionCommandService) SagaSubPosition(ctx context.Context, barrier any, userID, symbol string, quantity decimal.Decimal) error {
	return s.repo.ExecWithBarrier(ctx, barrier, func(ctx context.Context) error {
		pos, err := s.repo.GetByUserSymbol(ctx, userID, symbol)
		if err != nil || pos == nil {
			return nil
		}
		pos.Quantity -= quantity.InexactFloat64()
		return s.repo.Update(ctx, pos)
	})
}
