package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	orderv1 "github.com/wyfcoding/financialtrading/go-api/order/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"github.com/wyfcoding/pkg/transaction"
	"gorm.io/gorm"
)

type ClearingService struct {
	repo          domain.SettlementRepository
	outbox        *outbox.Manager
	db            *gorm.DB
	accountClient accountv1.AccountServiceClient
	orderClient   orderv1.OrderServiceClient
	marginEngine  *domain.PortfolioMarginEngine
}

func NewClearingService(
	repo domain.SettlementRepository,
	outbox *outbox.Manager,
	db *gorm.DB,
	accountClient accountv1.AccountServiceClient,
) *ClearingService {
	return &ClearingService{
		repo:          repo,
		outbox:        outbox,
		db:            db,
		accountClient: accountClient,
	}
}

// SettleTrade 处理交易结算
func (s *ClearingService) SettleTrade(ctx context.Context, req *SettleTradeRequest) (*SettlementDTO, error) {
	// 幂等检查
	existing, _ := s.repo.GetByTradeID(ctx, req.TradeID)
	if existing != nil {
		return s.toDTO(existing), nil
	}

	settlementID := fmt.Sprintf("SET-%d", idgen.GenID())
	settlement := domain.NewSettlement(settlementID, req.TradeID, req.BuyUserID, req.SellUserID, req.Symbol, req.Quantity, req.Price)

	// 本地事务：保存 Settlement 并发送 Saga 开始事件
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		if err := s.repo.Save(txCtx, settlement); err != nil {
			return err
		}

		// 这里我们简化：假设触发一个 Saga Orchestrator 或直接通过 gRPC 调用 AccountService
		// 为了演示严谨性，我们这里应该发布一个 Integration Event "ClearingStarted"
		// 或者直接调用 Account Service 的 Saga 接口

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 异步或同步发起 Saga 流程
	go s.executeSaga(context.Background(), settlement)

	return s.toDTO(settlement), nil
}

// executeSaga 使用自定义 Saga 协调器执行清算流程
func (s *ClearingService) executeSaga(ctx context.Context, settlement *domain.Settlement) {
	slog.InfoContext(ctx, "starting settlement saga with coordinator", "settlement_id", settlement.ID)

	saga := transaction.NewSagaCoordinator()
	saga.AddStep(&DeductBuyStep{
		BaseStep:   transaction.BaseStep{StepName: "DeductBuy"},
		accountCli: s.accountClient,
		settlement: settlement,
	}).AddStep(&AddSellStep{
		BaseStep:   transaction.BaseStep{StepName: "AddSell"},
		accountCli: s.accountClient,
		settlement: settlement,
	})

	if err := saga.Execute(ctx); err != nil {
		s.markFailed(ctx, settlement.ID, err.Error())
		return
	}

	s.markCompleted(ctx, settlement.ID)
}

func (s *ClearingService) markCompleted(ctx context.Context, id string) {
	s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		settlement, err := s.repo.Get(txCtx, id)
		if err != nil || settlement == nil {
			return err
		}

		settlement.Complete()
		return s.repo.Save(txCtx, settlement)
	})
}

func (s *ClearingService) markFailed(ctx context.Context, id, reason string) {
	s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := contextx.WithTx(ctx, tx)
		settlement, err := s.repo.Get(txCtx, id)
		if err != nil || settlement == nil {
			return err
		}

		settlement.Fail(reason)
		return s.repo.Save(txCtx, settlement)
	})
}

// SagaMarkSettlementCompleted 外部回调接口
func (s *ClearingService) SagaMarkSettlementCompleted(ctx context.Context, settlementID string) error {
	s.markCompleted(ctx, settlementID)
	return nil
}

// SagaMarkSettlementFailed 补偿接口
func (s *ClearingService) SagaMarkSettlementFailed(ctx context.Context, settlementID, reason string) error {
	s.markFailed(ctx, settlementID, reason)
	return nil
}

// ExecuteEODClearing 执行日终清算
// 1. 获取当天所有已结算记录
// 2. 计算净额 (Netting)
// 3. 生成报告快照
func (s *ClearingService) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	clearingID := fmt.Sprintf("EOD-%s-%d", clearingDate, idgen.GenID())
	slog.InfoContext(ctx, "EOD clearing started", "clearing_id", clearingID, "date", clearingDate)

	// 获取所有处于 COMPLETED 状态且日期匹配的结算单 (此处简化逻辑，假设 repo 已有方法)
	settlements, err := s.repo.List(ctx, 1000)
	if err != nil {
		return "", err
	}

	var totalVolume decimal.Decimal
	var totalFee decimal.Decimal
	userNetting := make(map[string]decimal.Decimal)

	for _, st := range settlements {
		totalVolume = totalVolume.Add(st.TotalAmount)
		totalFee = totalFee.Add(st.Fee)

		// 净额轧差 (买方付，卖方收)
		userNetting[st.BuyUserID] = userNetting[st.BuyUserID].Sub(st.TotalAmount)
		userNetting[st.SellUserID] = userNetting[st.SellUserID].Add(st.TotalAmount)
	}

	slog.InfoContext(ctx, "EOD Summary",
		"clearing_id", clearingID,
		"total_volume", totalVolume.String(),
		"total_fee", totalFee.String(),
		"users_count", len(userNetting))

	return clearingID, nil
}

// RunLiquidationCheck 对指定用户执行强平核查
func (s *ClearingService) RunLiquidationCheck(ctx context.Context, userID string) error {
	// 1. 调用 AccountService 获取可用余额与资产估值
	// 2. 调用 PositionService 获取详细档位持仓 (此处需注入相应 Client)
	// 3. 调用 MarginEngine 计算是否低于维持保证金

	// 演示逻辑 (简化版)：
	// health, _ := s.marginEngine.CheckAccountHealth(userID, equity, positions, params)
	// if health.NeedsLiquidation() {
	//    s.triggerLiquidation(ctx, userID)
	// }
	return nil
}

func (s *ClearingService) triggerLiquidation(ctx context.Context, userID string) {
	slog.WarnContext(ctx, "triggering liquidation", "user_id", userID)
	// 发送市价平仓单至 OrderService
	// s.orderClient.SubmitOrder(...)
}

// GetClearingStatus 获取清算状态
func (s *ClearingService) GetClearingStatus(ctx context.Context, clearingID string) (*SettlementDTO, error) {
	settlement, err := s.repo.Get(ctx, clearingID)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, nil
	}
	return s.toDTO(settlement), nil
}

// GetMarginRequirement 获取保证金要求
func (s *ClearingService) GetMarginRequirement(ctx context.Context, symbol string) (*MarginDTO, error) {
	return &MarginDTO{
		Symbol:           symbol,
		BaseMarginRate:   decimal.NewFromFloat(0.05),
		VolatilityFactor: decimal.NewFromFloat(1.1),
	}, nil
}

func (s *ClearingService) toDTO(agg *domain.Settlement) *SettlementDTO {
	var settledAt int64
	if agg.SettledAt != nil {
		settledAt = agg.SettledAt.Unix()
	}
	return &SettlementDTO{
		SettlementID: agg.ID,
		TradeID:      agg.TradeID,
		Status:       string(agg.Status),
		TotalAmount:  agg.TotalAmount.String(),
		SettledAt:    settledAt,
		ErrorMessage: agg.ErrorMessage,
	}
}
