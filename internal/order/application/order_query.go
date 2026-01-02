package application

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// OrderQuery 处理所有订单相关的查询操作（Queries）。
type OrderQuery struct {
	repo domain.OrderRepository
}

// NewOrderQuery 构造函数。
func NewOrderQuery(repo domain.OrderRepository) *OrderQuery {
	return &OrderQuery{repo: repo}
}

// GetOrder 获取订单详情
func (q *OrderQuery) GetOrder(ctx context.Context, orderID, userID string) (*OrderDTO, error) {
	order, err := q.repo.Get(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("order does not belong to user: %s", userID)
	}

	return &OrderDTO{
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		OrderType:      string(order.Type),
		Price:          order.Price.String(),
		Quantity:       order.Quantity.String(),
		FilledQuantity: order.FilledQuantity.String(),
		Status:         string(order.Status),
		TimeInForce:    string(order.TimeInForce),
		CreatedAt:      order.CreatedAt.Unix(),
		UpdatedAt:      order.UpdatedAt.Unix(),
	}, nil
}

// ListOrders 分页列出指定用户的订单记录。
func (q *OrderQuery) ListOrders(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	orders, total, err := q.repo.ListByUser(ctx, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*OrderDTO, len(orders))
	for i, o := range orders {
		dtos[i] = &OrderDTO{
			OrderID:        o.OrderID,
			UserID:         o.UserID,
			Symbol:         o.Symbol,
			Side:           string(o.Side),
			OrderType:      string(o.Type),
			Price:          o.Price.String(),
			Quantity:       o.Quantity.String(),
			FilledQuantity: o.FilledQuantity.String(),
			Status:         string(o.Status),
			TimeInForce:    string(o.TimeInForce),
			CreatedAt:      o.CreatedAt.Unix(),
			UpdatedAt:      o.UpdatedAt.Unix(),
		}
	}
	return dtos, total, nil
}
