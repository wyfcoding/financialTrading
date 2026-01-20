package application

import (
	"context"
	"fmt"

	accountv1 "github.com/wyfcoding/financialtrading/go-api/account/v1"
	riskv1 "github.com/wyfcoding/financialtrading/go-api/risk/v1"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/transaction"
)

// OrderCreateStep 订单创建步骤
type OrderCreateStep struct {
	transaction.BaseStep
	repo  domain.OrderRepository
	order *domain.Order
}

func (s *OrderCreateStep) Execute(ctx context.Context) error {
	s.order.Status = domain.StatusPending
	return s.repo.Save(ctx, s.order)
}

func (s *OrderCreateStep) Compensate(ctx context.Context) error {
	return s.repo.UpdateStatus(ctx, s.order.ID, domain.StatusCancelled)
}

// RiskCheckStep 风控校验与保证金锁定步骤
type RiskCheckStep struct {
	transaction.BaseStep
	riskCli riskv1.RiskServiceClient
	order   *domain.Order
}

func (s *RiskCheckStep) Execute(ctx context.Context) error {
	if s.riskCli == nil {
		return nil
	}
	resp, err := s.riskCli.CheckRisk(ctx, &riskv1.CheckRiskRequest{
		UserId:   s.order.UserID,
		Symbol:   s.order.Symbol,
		Side:     string(s.order.Side),
		Quantity: s.order.Quantity,
		Price:    s.order.Price,
	})
	if err != nil {
		return err
	}
	if !resp.Passed {
		return fmt.Errorf("risk check failed: %s", resp.Reason)
	}
	return nil
}

func (s *RiskCheckStep) Compensate(ctx context.Context) error {
	// 补偿逻辑：如果之前由于保证金锁定，这里应释放锁定
	return nil
}

// AccountBalanceStep 账户余额扣减步骤
type AccountBalanceStep struct {
	transaction.BaseStep
	accountCli accountv1.AccountServiceClient
	order      *domain.Order
	amount     string
}

func (s *AccountBalanceStep) Execute(ctx context.Context) error {
	if s.accountCli == nil {
		return nil
	}
	// 实际应调用正式的扣减接口，此处模拟
	return nil
}

func (s *AccountBalanceStep) Compensate(ctx context.Context) error {
	// 补偿逻辑：退回资金
	return nil
}
