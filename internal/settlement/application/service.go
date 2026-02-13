package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

type SettlementAppService struct {
	repo   domain.SettlementRepository
	logger *slog.Logger
}

func NewSettlementAppService(repo domain.SettlementRepository, logger *slog.Logger) *SettlementAppService {
	return &SettlementAppService{
		repo:   repo,
		logger: logger,
	}
}

// CreateInstructionCommand 创建结算指令命令
type CreateInstructionCommand struct {
	TradeID   string
	Symbol    string
	Quantity  int64
	Price     float64
	BuyerID   string
	SellerID  string
	Currency  string
	CycleDays int
}

func (s *SettlementAppService) CreateInstruction(ctx context.Context, cmd CreateInstructionCommand) (*domain.SettlementInstruction, error) {
	now := time.Now()
	// T+N Logic
	settleDate := now.AddDate(0, 0, cmd.CycleDays)

	instruction := &domain.SettlementInstruction{
		InstructionID: fmt.Sprintf("INS-%s-%d", cmd.TradeID, now.UnixNano()),
		TradeID:       cmd.TradeID,
		Symbol:        cmd.Symbol,
		Quantity:      cmd.Quantity,
		Price:         cmd.Price,
		Amount:        float64(cmd.Quantity) * cmd.Price,
		Currency:      cmd.Currency,
		BuyerAccount:  cmd.BuyerID,
		SellerAccount: cmd.SellerID,
		TradeDate:     now,
		SettleDate:    settleDate,
		Status:        domain.StatusPending,
	}

	if err := s.repo.Save(ctx, instruction); err != nil {
		return nil, fmt.Errorf("failed to save instruction: %w", err)
	}

	s.logger.InfoContext(ctx, "settlement instruction created", "id", instruction.InstructionID, "settle_date", settleDate)
	return instruction, nil
}

func (s *SettlementAppService) GetInstruction(ctx context.Context, id string) (*domain.SettlementInstruction, error) {
	return s.repo.Get(ctx, id)
}

// BatchSettle 执行批量结算 (简化版：直接标记完成)
func (s *SettlementAppService) BatchSettle(ctx context.Context, targetDate time.Time) (int, int, int, []string, error) {
	// Find pending instructions
	instructions, err := s.repo.FindPendingByDate(ctx, targetDate, 1000)
	if err != nil {
		return 0, 0, 0, nil, err
	}

	processed := 0
	success := 0
	failed := 0
	var failedIDs []string

	for _, ins := range instructions {
		processed++
		// Mock Settlement Logic:
		// 1. Verify Buyer has Cash
		// 2. Verify Seller has Securities
		// 3. Move Assets (Call Account/Custody Service)
		// Here we just mark as Settled for skeletal implementation

		err := s.repo.UpdateStatus(ctx, ins.InstructionID, domain.StatusSettled, "")
		if err != nil {
			failed++
			failedIDs = append(failedIDs, ins.InstructionID)
			s.logger.ErrorContext(ctx, "failed to settle instruction", "id", ins.InstructionID, "error", err)
		} else {
			success++
		}
	}

	return processed, success, failed, failedIDs, nil
}
