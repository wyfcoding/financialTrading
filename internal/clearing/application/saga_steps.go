package application

import (
	"context"

	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/domain"
	"github.com/wyfcoding/pkg/transaction"
)

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
