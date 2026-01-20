package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

type OrderQuery struct {
	repo domain.OrderRepository
}

func NewOrderQuery(repo domain.OrderRepository) *OrderQuery {
	return &OrderQuery{repo: repo}
}

func (q *OrderQuery) GetOrder(ctx context.Context, orderID string) (*OrderDTO, error) {
	order, err := q.repo.Get(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	return &OrderDTO{
		OrderID:        order.ID,
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          decimal.NewFromFloat(order.Price).String(),
		Quantity:       decimal.NewFromFloat(order.Quantity).String(),
		FilledQuantity: decimal.NewFromFloat(order.FilledQuantity).String(),
		Status:         string(order.Status),
		// TimeInForce removed
		CreatedAt: order.CreatedAt.Unix(),
		UpdatedAt: order.UpdatedAt.Unix(),
	}, nil
}
