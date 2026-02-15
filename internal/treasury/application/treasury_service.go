// Package application 资金管理核心服务
// 生成摘要：
// 1) 整合 CashPool、LiquidityForecast、TransferInstruction 的业务逻辑
// 2) 负责资金池水位监控与自动调拨建议
// 3) 生成流动性缺口分析报告
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/treasury/domain"
	"github.com/wyfcoding/pkg/idgen"
)

// TreasuryService 资金管理服务
type TreasuryService struct {
	cashPoolRepo     domain.CashPoolRepository
	liquidityRepo    domain.LiquidityForecastRepository
	transferInsRepo  domain.TransferInstructionRepository
	accountRepo      domain.AccountRepository     // 底层物理账户操作
	transactionRepo  domain.TransactionRepository // 交易流水记录
	commandService   *CommandService              // 复用现有的资金命令服务（如转账）
	logger           *slog.Logger
}

func NewTreasuryService(
	cashPoolRepo domain.CashPoolRepository,
	liquidityRepo domain.LiquidityForecastRepository,
	transferInsRepo domain.TransferInstructionRepository,
	accountRepo domain.AccountRepository,
	transactionRepo domain.TransactionRepository,
	commandService *CommandService,
	logger *slog.Logger,
) *TreasuryService {
	return &TreasuryService{
		cashPoolRepo:    cashPoolRepo,
		liquidityRepo:   liquidityRepo,
		transferInsRepo: transferInsRepo,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		commandService:  commandService,
		logger:          logger.With("module", "treasury_service"),
	}
}

// CreateCashPool 创建资金池
func (s *TreasuryService) CreateCashPool(ctx context.Context, name, currency string, min, max decimal.Decimal) (*domain.CashPool, error) {
	pool := domain.NewCashPool(name, currency, min, max)
	if err := s.cashPoolRepo.Save(ctx, pool); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "cash pool created", "pool_name", name)
	return pool, nil
}

// AddBankAccountToPool 添加银行账户到资金池
func (s *TreasuryService) AddBankAccountToPool(ctx context.Context, poolID uint64, bankName, accountNo, accountName, swiftCode, currency string) error {
	pool, err := s.cashPoolRepo.GetByID(ctx, poolID)
	if err != nil {
		return err
	}

	account := domain.BankAccount{
		PoolID:      poolID,
		BankName:    bankName,
		AccountNo:   accountNo,
		AccountName: accountName,
		SwiftCode:   swiftCode,
		Currency:    currency,
		Balance:     decimal.Zero,
		Status:      domain.BankAccountStatusActive,
	}

	if err := pool.AddAccount(account); err != nil {
		return err
	}

	if err := s.cashPoolRepo.Save(ctx, pool); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "bank account added to pool", "pool_id", poolID, "account_no", accountNo)
	return nil
}

// SyncAccountBalance 同步银行账户余额（并在内部逻辑账户上反映）
// 这是一个模拟实现，实际应调用银企直连网关
func (s *TreasuryService) SyncAccountBalance(ctx context.Context, accountID uint, newBalance decimal.Decimal) error {
	// TODO: 1. 找到账户所属资金池
	// TODO: 2. 更新 bank_account 余额
	// TODO: 3. 更新 pool.total_balance
	// TODO: 4. 如果变动金额大，可能生成一条内部 Transaction 记录
	return errors.New("not implemented")
}

// MonitorLiquidity 监控资金池水位，生成调拨建议
func (s *TreasuryService) MonitorLiquidity(ctx context.Context, poolID uint64) error {
	pool, err := s.cashPoolRepo.GetByID(ctx, poolID)
	if err != nil {
		return err
	}

	isLow, isHigh, deviation := pool.CheckLiquidity()
	if isLow {
		s.logger.WarnContext(ctx, "cash pool liquidity low", "pool_name", pool.Name, "deviation", deviation)
		// 自动生成调拨建议（从主账户或其他池调入）
		// TODO: Create TransferInstruction Proposal
	}
	if isHigh {
		s.logger.InfoContext(ctx, "cash pool liquidity high", "pool_name", pool.Name, "excess", deviation)
		// 自动生成投资建议（归集到主账户或理财）
	}
	return nil
}

// CreateLiquidityForecast 创建流动性预测条目
func (s *TreasuryService) CreateLiquidityForecast(ctx context.Context, poolID uint64, date time.Time, typ domain.ForecastType, amount decimal.Decimal, curr string) error {
	forecast := &domain.LiquidityForecast{
		PoolID:     poolID,
		Date:       date,
		Type:       typ,
		Amount:     amount,
		Currency:   curr,
		Confidence: domain.ConfidenceLevelMedium,
		Status:     "PENDING",
	}
	return s.liquidityRepo.Save(ctx, forecast)
}

// AnalyzeLiquidityGap 分析未来 N 天的流动性缺口
func (s *TreasuryService) AnalyzeLiquidityGap(ctx context.Context, poolID uint64, days int) ([]domain.LiquidityGap, error) {
	pool, err := s.cashPoolRepo.GetByID(ctx, poolID)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	end := start.AddDate(0, 0, days)
	forecasts, err := s.liquidityRepo.ListByPoolAndDateRange(ctx, poolID, start, end)
	if err != nil {
		return nil, err
	}

	// 聚合计算
	gaps := make([]domain.LiquidityGap, 0, days)
	currentBalance := pool.TotalBalance

	// 按日期分组
	dailyMap := make(map[string][]*domain.LiquidityForecast)
	for _, f := range forecasts {
		day := f.Date.Format("2006-01-02")
		dailyMap[day] = append(dailyMap[day], f)
	}

	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i)
		dayKey := date.Format("2006-01-02")
		
		dailyItems := dailyMap[dayKey]
		inflow := decimal.Zero
		outflow := decimal.Zero

		for _, item := range dailyItems {
			if item.Type == domain.ForecastTypeInflow {
				inflow = inflow.Add(item.Amount)
			} else {
				outflow = outflow.Add(item.Amount)
			}
		}

		net := inflow.Sub(outflow)
		opening := currentBalance
		closing := currentBalance.Add(net)
		gap := closing.Sub(pool.MinTarget) // 缺口 = 余额 - 最低目标

		gaps = append(gaps, domain.LiquidityGap{
			Date:            date,
			OpeningBalance:  opening,
			ProjectedInflow: inflow,
			ProjectedOutflow: outflow,
			NetCashFlow:     net,
			ClosingBalance:  closing,
			Gap:             gap,
		})

		currentBalance = closing
	}

	return gaps, nil
}

// InitiateTransfer 发起资金调拨
func (s *TreasuryService) InitiateTransfer(ctx context.Context, fromAccountID, toAccountID uint64, amount decimal.Decimal, curr, purpose string) (string, error) {
	insID := fmt.Sprintf("INS%s", idgen.GenIDString())
	ins := &domain.TransferInstruction{
		InstructionID: insID,
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Currency:      curr,
		RequestDate:   time.Now(),
		Status:        domain.InstructionStatusPending,
		Purpose:       purpose,
	}

	if err := s.transferInsRepo.Save(ctx, ins); err != nil {
		return "", err
	}
	return insID, nil
}

// ApproveTransfer 审批并执行调拨
func (s *TreasuryService) ApproveTransfer(ctx context.Context, insID string, approverID string) error {
	ins, err := s.transferInsRepo.GetByID(ctx, insID)
	if err != nil {
		return err
	}

	if err := ins.Approve(approverID); err != nil {
		return err
	}

	// 立即执行
	if err := ins.Execute(); err != nil {
		return err
	}
	s.transferInsRepo.Save(ctx, ins) // 保存中间状态

	// 调用 CommandService 执行底层账户转账（使用分）
	amountCents := ins.Amount.Mul(decimal.NewFromInt(100)).IntPart()
	txID, err := s.commandService.Transfer(ctx, TransferCommand{
		FromAccountID: ins.FromAccountID,
		ToAccountID:   ins.ToAccountID,
		Amount:        amountCents,
		RefID:         ins.InstructionID,
		Remark:        fmt.Sprintf("Treasury Transfer: %s", ins.Purpose),
	})

	if err != nil {
		ins.Fail(err.Error())
		s.transferInsRepo.Save(ctx, ins)
		return err
	}

	ins.Complete(txID)
	return s.transferInsRepo.Save(ctx, ins)
}
