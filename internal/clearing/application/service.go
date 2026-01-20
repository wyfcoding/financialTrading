package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/messagequeue/outbox"
	"gorm.io/gorm"
)

type ClearingService struct {
	repo          domain.SettlementRepository
	outbox        *outbox.Manager
	db            *gorm.DB
	accountClient accountv1.AccountServiceClient
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

// executeSaga 是一个简单的 Saga 编排器 (实际生产应使用 DTM 或 Cadence)
func (s *ClearingService) executeSaga(ctx context.Context, settlement *domain.Settlement) {

	slog.InfoContext(ctx, "starting settlement saga", "settlement_id", settlement.ID)

	totalAmount := settlement.TotalAmount.String()
	currency := "USDT" // 假设结算币种

	// Step 1: 扣除买方冻结资金
	_, err := s.accountClient.SagaDeductFrozen(ctx, &accountv1.SagaAccountRequest{
		UserId:   settlement.BuyUserID,
		Currency: currency,
		Amount:   totalAmount,
		TradeId:  settlement.TradeID,
	})
	if err != nil {
		s.markFailed(ctx, settlement.ID, "failed to deduct buy user fund: "+err.Error())
		return
	}

	// Step 2: 增加卖方余额
	_, err = s.accountClient.SagaAddBalance(ctx, &accountv1.SagaAccountRequest{
		UserId:   settlement.SellUserID,
		Currency: currency,
		Amount:   totalAmount,
		TradeId:  settlement.TradeID,
	})
	if err != nil {
		// 补偿 Step 1: 退还买方资金
		s.accountClient.SagaRefundFrozen(ctx, &accountv1.SagaAccountRequest{
			UserId:   settlement.BuyUserID,
			Currency: currency,
			Amount:   totalAmount,
			TradeId:  settlement.TradeID,
		})
		s.markFailed(ctx, settlement.ID, "failed to add sell user balance: "+err.Error())
		return
	}

	// Step 3: 完成
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
func (s *ClearingService) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	clearingID := fmt.Sprintf("EOD-%s-%d", clearingDate, idgen.GenID())
	slog.InfoContext(ctx, "EOD clearing started", "clearing_id", clearingID, "date", clearingDate)
	// 实现在此处添加逻辑
	return clearingID, nil
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
