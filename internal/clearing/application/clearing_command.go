package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/contextx"
	"github.com/wyfcoding/pkg/idgen"
	"github.com/wyfcoding/pkg/transaction"
)

type SettleTradeRequest struct {
	TradeID    string
	BuyUserID  string
	SellUserID string
	Symbol     string
	Quantity   decimal.Decimal
	Price      decimal.Decimal
}

type SettleTradeCommand = SettleTradeRequest

type MarkSettlementCommand struct {
	SettlementID string
	Reason       string
	Success      bool
}

// DeductBuyStep 扣除买方资金步骤
type DeductBuyStep struct {
	transaction.BaseStep
	accountCli accountv1.AccountServiceClient
	settlement *domain.Settlement
}

func (s *DeductBuyStep) Execute(ctx context.Context) error {
	_, err := s.accountCli.SagaDeductFrozen(ctx, &accountv1.SagaAccountRequest{
		UserId:   s.settlement.BuyUserID,
		Currency: "USDT",
		Amount:   s.settlement.TotalAmount.String(),
		TradeId:  s.settlement.TradeID,
	})
	return err
}

func (s *DeductBuyStep) Compensate(ctx context.Context) error {
	_, err := s.accountCli.SagaRefundFrozen(ctx, &accountv1.SagaAccountRequest{
		UserId:   s.settlement.BuyUserID,
		Currency: "USDT",
		Amount:   s.settlement.TotalAmount.String(),
		TradeId:  s.settlement.TradeID,
	})
	return err
}

// AddSellStep 增加卖方资金步骤
type AddSellStep struct {
	transaction.BaseStep
	accountCli accountv1.AccountServiceClient
	settlement *domain.Settlement
}

func (s *AddSellStep) Execute(ctx context.Context) error {
	_, err := s.accountCli.SagaAddBalance(ctx, &accountv1.SagaAccountRequest{
		UserId:   s.settlement.SellUserID,
		Currency: "USDT",
		Amount:   s.settlement.TotalAmount.String(),
		TradeId:  s.settlement.TradeID,
	})
	return err
}

func (s *AddSellStep) Compensate(ctx context.Context) error {
	// 补偿：如果之前加钱了，现在要扣回来 (实际生产中需配合冲正接口)
	return nil
}

// ClearingCommandService 处理清算相关的写操作。
type ClearingCommandService struct {
	repo          domain.SettlementRepository
	redisRepo     domain.MarginRedisRepository
	publisher     domain.EventPublisher
	accountClient accountv1.AccountServiceClient
}

func NewClearingCommandService(
	repo domain.SettlementRepository,
	redisRepo domain.MarginRedisRepository,
	publisher domain.EventPublisher,
	accountClient accountv1.AccountServiceClient,
) *ClearingCommandService {
	return &ClearingCommandService{
		repo:          repo,
		redisRepo:     redisRepo,
		publisher:     publisher,
		accountClient: accountClient,
	}
}

// SettleTrade 处理交易结算
func (s *ClearingCommandService) SettleTrade(ctx context.Context, req *SettleTradeRequest) (*SettlementDTO, error) {
	// 幂等检查
	existing, _ := s.repo.GetByTradeID(ctx, req.TradeID)
	if existing != nil {
		return s.toDTO(existing), nil
	}

	settlementID := fmt.Sprintf("SET-%d", idgen.GenID())
	settlement := domain.NewSettlement(settlementID, req.TradeID, req.BuyUserID, req.SellUserID, req.Symbol, req.Quantity, req.Price)

	// 本地事务：保存 Settlement 并发送 Saga 开始事件
	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Save(txCtx, settlement); err != nil {
			return err
		}

		if s.publisher == nil {
			return nil
		}
		// 发送集成事件 (Outbox Pattern)
		event := domain.SettlementCreatedEvent{
			BaseEvent:    domain.BaseEvent{Timestamp: time.Now()},
			SettlementID: settlementID,
			TradeID:      req.TradeID,
			TotalAmount:  settlement.TotalAmount.String(),
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SettlementCreatedEventType, settlementID, event)
	})
	if err != nil {
		return nil, err
	}

	// 异步或同步发起 Saga 流程
	go s.executeSaga(context.Background(), settlement)

	return s.toDTO(settlement), nil
}

// executeSaga 使用自定义 Saga 协调器执行清算流程
func (s *ClearingCommandService) executeSaga(ctx context.Context, settlement *domain.Settlement) {
	slog.InfoContext(ctx, "starting settlement saga with coordinator", "settlement_id", settlement.SettlementID)

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
		s.markFailed(ctx, settlement.SettlementID, err.Error())
		return
	}

	s.markCompleted(ctx, settlement.SettlementID)
}

func (s *ClearingCommandService) markCompleted(ctx context.Context, id string) {
	_ = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		settlement, err := s.repo.Get(txCtx, id)
		if err != nil || settlement == nil {
			return err
		}

		settlement.Complete()
		if err := s.repo.Save(txCtx, settlement); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		event := domain.SettlementCompletedEvent{
			BaseEvent:    domain.BaseEvent{Timestamp: time.Now()},
			SettlementID: settlement.SettlementID,
			TradeID:      settlement.TradeID,
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SettlementCompletedEventType, settlement.SettlementID, event)
	})
}

func (s *ClearingCommandService) markFailed(ctx context.Context, id, reason string) {
	_ = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		settlement, err := s.repo.Get(txCtx, id)
		if err != nil || settlement == nil {
			return err
		}

		settlement.Fail(reason)
		if err := s.repo.Save(txCtx, settlement); err != nil {
			return err
		}
		if s.publisher == nil {
			return nil
		}
		event := domain.SettlementFailedEvent{
			BaseEvent:    domain.BaseEvent{Timestamp: time.Now()},
			SettlementID: settlement.SettlementID,
			TradeID:      settlement.TradeID,
			Reason:       reason,
		}
		return s.publisher.PublishInTx(ctx, contextx.GetTx(txCtx), domain.SettlementFailedEventType, settlement.SettlementID, event)
	})
}

// SagaMarkSettlementCompleted 外部回调接口
func (s *ClearingCommandService) SagaMarkSettlementCompleted(ctx context.Context, settlementID string) error {
	s.markCompleted(ctx, settlementID)
	return nil
}

// SagaMarkSettlementFailed 补偿接口
func (s *ClearingCommandService) SagaMarkSettlementFailed(ctx context.Context, settlementID, reason string) error {
	s.markFailed(ctx, settlementID, reason)
	return nil
}

// ExecuteEODClearing 执行日终清算
func (s *ClearingCommandService) ExecuteEODClearing(ctx context.Context, clearingDate string) (string, error) {
	clearingID := fmt.Sprintf("EOD-%s-%d", clearingDate, idgen.GenID())
	slog.InfoContext(ctx, "EOD clearing started", "clearing_id", clearingID, "date", clearingDate)

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
func (s *ClearingCommandService) RunLiquidationCheck(ctx context.Context, userID string) error {
	return nil
}

func (s *ClearingCommandService) toDTO(agg *domain.Settlement) *SettlementDTO {
	var settledAt int64
	if agg.SettledAt != nil {
		settledAt = agg.SettledAt.Unix()
	}
	return &SettlementDTO{
		SettlementID: agg.SettlementID,
		TradeID:      agg.TradeID,
		Status:       string(agg.Status),
		TotalAmount:  agg.TotalAmount.String(),
		SettledAt:    settledAt,
		ErrorMessage: agg.ErrorMessage,
	}
}
