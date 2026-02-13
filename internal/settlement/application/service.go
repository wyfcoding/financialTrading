package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

type SettlementAppService struct {
	repo          domain.SettlementRepository
	nettingRepo   domain.NettingRepository
	batchRepo     domain.BatchRepository
	fxRateRepo    domain.FXRateRepository
	domainSvc     *domain.SettlementDomainService
	logger        *slog.Logger
}

func NewSettlementAppService(
	repo domain.SettlementRepository,
	nettingRepo domain.NettingRepository,
	batchRepo domain.BatchRepository,
	fxRateRepo domain.FXRateRepository,
	domainSvc *domain.SettlementDomainService,
	logger *slog.Logger,
) *SettlementAppService {
	return &SettlementAppService{
		repo:        repo,
		nettingRepo: nettingRepo,
		batchRepo:   batchRepo,
		fxRateRepo:  fxRateRepo,
		domainSvc:   domainSvc,
		logger:      logger,
	}
}

type CreateInstructionCommand struct {
	TradeID        string
	OrderID        string
	Symbol         string
	SecurityType   string
	Quantity       float64
	Price          float64
	Currency       string
	BuyerAccountID string
	SellerAccountID string
	SettlementType int
	CycleDays      int
	CCPFlag        bool
	CCPAccount     string
}

func (s *SettlementAppService) CreateInstruction(ctx context.Context, cmd CreateInstructionCommand) (*domain.SettlementInstruction, error) {
	quantity := decimal.NewFromFloat(cmd.Quantity)
	price := decimal.NewFromFloat(cmd.Price)
	
	instruction := domain.NewSettlementInstruction(
		cmd.TradeID,
		cmd.Symbol,
		quantity,
		price,
		cmd.Currency,
		cmd.BuyerAccountID,
		cmd.SellerAccountID,
		cmd.CycleDays,
	)
	
	if cmd.OrderID != "" {
		instruction.OrderID = cmd.OrderID
	}
	if cmd.SecurityType != "" {
		instruction.SecurityType = cmd.SecurityType
	}
	if cmd.SettlementType > 0 {
		instruction.SettlementType = domain.SettlementType(cmd.SettlementType)
	}
	if cmd.CCPFlag {
		instruction.SetCCP(cmd.CCPAccount)
	}
	
	if err := s.repo.Save(ctx, instruction); err != nil {
		return nil, fmt.Errorf("failed to save instruction: %w", err)
	}
	
	s.logger.InfoContext(ctx, "settlement instruction created",
		"instruction_id", instruction.InstructionID,
		"trade_id", cmd.TradeID,
		"settlement_date", instruction.SettlementDate,
	)
	
	return instruction, nil
}

type SetCustodianCommand struct {
	InstructionID    string
	BuyerCustodian   string
	BuyerSettleAcct  string
	SellerCustodian  string
	SellerSettleAcct string
}

func (s *SettlementAppService) SetCustodian(ctx context.Context, cmd SetCustodianCommand) error {
	instruction, err := s.repo.Get(ctx, cmd.InstructionID)
	if err != nil {
		return fmt.Errorf("failed to get instruction: %w", err)
	}
	
	instruction.SetCustodian(cmd.BuyerCustodian, cmd.BuyerSettleAcct, cmd.SellerCustodian, cmd.SellerSettleAcct)
	
	if err := s.repo.Update(ctx, instruction); err != nil {
		return fmt.Errorf("failed to update instruction: %w", err)
	}
	
	s.logger.InfoContext(ctx, "custodian set", "instruction_id", cmd.InstructionID)
	return nil
}

func (s *SettlementAppService) GetInstruction(ctx context.Context, instructionID string) (*domain.SettlementInstruction, error) {
	return s.repo.Get(ctx, instructionID)
}

func (s *SettlementAppService) GetInstructionByTradeID(ctx context.Context, tradeID string) (*domain.SettlementInstruction, error) {
	return s.repo.GetByTradeID(ctx, tradeID)
}

type ProcessSettlementCommand struct {
	InstructionID string
	BatchID       string
}

func (s *SettlementAppService) ProcessSettlement(ctx context.Context, cmd ProcessSettlementCommand) error {
	instruction, err := s.repo.Get(ctx, cmd.InstructionID)
	if err != nil {
		return fmt.Errorf("failed to get instruction: %w", err)
	}
	
	if instruction.Status == domain.SettlementStatusCleared {
		if err := instruction.StartProcessing(cmd.BatchID); err != nil {
			return err
		}
	} else if instruction.Status == domain.SettlementStatusPending {
		if err := instruction.StartProcessing(cmd.BatchID); err != nil {
			return err
		}
	}
	
	if s.domainSvc != nil {
		if err := s.domainSvc.ValidateBalance(ctx, instruction); err != nil {
			instruction.Fail(fmt.Sprintf("balance validation failed: %v", err))
			_ = s.repo.Update(ctx, instruction)
			return err
		}
		
		if instruction.SettlementType == domain.SettlementTypeDVP {
			if err := s.domainSvc.ExecuteDVP(ctx, instruction); err != nil {
				instruction.Fail(fmt.Sprintf("DVP execution failed: %v", err))
				_ = s.repo.Update(ctx, instruction)
				return err
			}
		}
	}
	
	if err := instruction.Settle(); err != nil {
		return err
	}
	
	if err := s.repo.Update(ctx, instruction); err != nil {
		return fmt.Errorf("failed to update instruction: %w", err)
	}
	
	if s.domainSvc != nil {
		_ = s.domainSvc.NotifyCompletion(ctx, instruction)
	}
	
	s.logger.InfoContext(ctx, "settlement completed", "instruction_id", cmd.InstructionID)
	return nil
}

type RetrySettlementCommand struct {
	InstructionID string
}

func (s *SettlementAppService) RetrySettlement(ctx context.Context, cmd RetrySettlementCommand) error {
	instruction, err := s.repo.Get(ctx, cmd.InstructionID)
	if err != nil {
		return fmt.Errorf("failed to get instruction: %w", err)
	}
	
	if !instruction.CanRetry() {
		return fmt.Errorf("instruction cannot be retried")
	}
	
	if err := instruction.Retry(); err != nil {
		return err
	}
	
	if err := s.repo.Update(ctx, instruction); err != nil {
		return fmt.Errorf("failed to update instruction: %w", err)
	}
	
	s.logger.InfoContext(ctx, "settlement retry", "instruction_id", cmd.InstructionID, "retry_count", instruction.RetryCount)
	return nil
}

type CancelSettlementCommand struct {
	InstructionID string
	Reason        string
}

func (s *SettlementAppService) CancelSettlement(ctx context.Context, cmd CancelSettlementCommand) error {
	instruction, err := s.repo.Get(ctx, cmd.InstructionID)
	if err != nil {
		return fmt.Errorf("failed to get instruction: %w", err)
	}
	
	if err := instruction.Cancel(cmd.Reason); err != nil {
		return err
	}
	
	if err := s.repo.Update(ctx, instruction); err != nil {
		return fmt.Errorf("failed to update instruction: %w", err)
	}
	
	s.logger.InfoContext(ctx, "settlement cancelled", "instruction_id", cmd.InstructionID, "reason", cmd.Reason)
	return nil
}

type BatchSettleCommand struct {
	SettlementDate time.Time
	BatchSize      int
}

type BatchSettleResult struct {
	BatchID      string
	TotalCount   int
	SuccessCount int
	FailedCount  int
	FailedIDs    []string
}

func (s *SettlementAppService) BatchSettle(ctx context.Context, cmd BatchSettleCommand) (*BatchSettleResult, error) {
	batchID := fmt.Sprintf("BATCH-%s", time.Now().Format("20060102150405"))
	
	batch := &domain.SettlementBatch{
		BatchID:        batchID,
		SettlementDate: cmd.SettlementDate,
		Status:         "PROCESSING",
	}
	now := time.Now()
	batch.StartedAt = &now
	
	if err := s.batchRepo.Save(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}
	
	instructions, err := s.repo.FindPendingByDate(ctx, cmd.SettlementDate, cmd.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending instructions: %w", err)
	}
	
	result := &BatchSettleResult{
		BatchID:    batchID,
		FailedIDs:  []string{},
	}
	
	for _, ins := range instructions {
		result.TotalCount++
		
		processCmd := ProcessSettlementCommand{
			InstructionID: ins.InstructionID,
			BatchID:       batchID,
		}
		
		if err := s.ProcessSettlement(ctx, processCmd); err != nil {
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, ins.InstructionID)
			s.logger.ErrorContext(ctx, "failed to settle instruction",
				"instruction_id", ins.InstructionID,
				"error", err,
			)
		} else {
			result.SuccessCount++
		}
	}
	
	batch.TotalCount = result.TotalCount
	batch.SuccessCount = result.SuccessCount
	batch.FailedCount = result.FailedCount
	batch.Status = "COMPLETED"
	completedAt := time.Now()
	batch.CompletedAt = &completedAt
	
	_ = s.batchRepo.Save(ctx, batch)
	
	s.logger.InfoContext(ctx, "batch settlement completed",
		"batch_id", batchID,
		"total", result.TotalCount,
		"success", result.SuccessCount,
		"failed", result.FailedCount,
	)
	
	return result, nil
}

type NettingCommand struct {
	AccountID string
	Currency  string
	Date      time.Time
}

func (s *SettlementAppService) PerformNetting(ctx context.Context, cmd NettingCommand) (*domain.NettingResult, error) {
	instructions, err := s.repo.FindPendingByAccount(ctx, cmd.AccountID, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to find instructions: %w", err)
	}
	
	nettingID := fmt.Sprintf("NET-%s-%d", cmd.AccountID[:8], time.Now().UnixNano())
	
	buyAmount := decimal.Zero
	sellAmount := decimal.Zero
	buyQty := decimal.Zero
	sellQty := decimal.Zero
	instructionIDs := []string{}
	
	for _, ins := range instructions {
		if ins.Currency != cmd.Currency {
			continue
		}
		
		if err := ins.StartNetting(nettingID); err != nil {
			continue
		}
		
		instructionIDs = append(instructionIDs, ins.InstructionID)
		
		if ins.BuyerAccountID == cmd.AccountID {
			buyAmount = buyAmount.Add(ins.Amount)
			buyQty = buyQty.Add(ins.Quantity)
		}
		if ins.SellerAccountID == cmd.AccountID {
			sellAmount = sellAmount.Add(ins.Amount)
			sellQty = sellQty.Add(ins.Quantity)
		}
		
		_ = ins.CompleteNetting()
		_ = s.repo.Update(ctx, ins)
	}
	
	result := &domain.NettingResult{
		NettingID:      nettingID,
		AccountID:      cmd.AccountID,
		Currency:       cmd.Currency,
		GrossAmount:    buyAmount.Add(sellAmount),
		NetAmount:      buyAmount.Sub(sellAmount),
		NetQuantity:    buyQty.Sub(sellQty),
		InstructionIDs: fmt.Sprintf("%v", instructionIDs),
		Status:         "COMPLETED",
		CreatedAt:      time.Now(),
	}
	
	if err := s.nettingRepo.Save(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save netting result: %w", err)
	}
	
	s.logger.InfoContext(ctx, "netting completed",
		"netting_id", nettingID,
		"account_id", cmd.AccountID,
		"net_amount", result.NetAmount,
	)
	
	return result, nil
}

type FXConvertCommand struct {
	FromCurrency string
	ToCurrency   string
	Amount       float64
}

type FXConvertResult struct {
	FromCurrency string
	ToCurrency   string
	FromAmount   float64
	ToAmount     float64
	Rate         float64
}

func (s *SettlementAppService) ConvertCurrency(ctx context.Context, cmd FXConvertCommand) (*FXConvertResult, error) {
	rate, err := s.fxRateRepo.GetRate(ctx, cmd.FromCurrency, cmd.ToCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to get FX rate: %w", err)
	}
	
	amount := decimal.NewFromFloat(cmd.Amount)
	convertedAmount := amount.Mul(rate.Rate)
	
	return &FXConvertResult{
		FromCurrency: cmd.FromCurrency,
		ToCurrency:   cmd.ToCurrency,
		FromAmount:   cmd.Amount,
		ToAmount:     convertedAmount.InexactFloat64(),
		Rate:         rate.Rate.InexactFloat64(),
	}, nil
}

type GetPendingInstructionsQuery struct {
	AccountID string
	Limit     int
}

func (s *SettlementAppService) GetPendingInstructions(ctx context.Context, query GetPendingInstructionsQuery) ([]*domain.SettlementInstruction, error) {
	if query.Limit <= 0 {
		query.Limit = 100
	}
	return s.repo.FindPendingByAccount(ctx, query.AccountID, query.Limit)
}
