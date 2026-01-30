package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// OrderQueryService 处理所有订单相关的查询操作（Queries）。
type OrderQueryService struct {
	repo domain.OrderRepository
}

// NewOrderQueryService 构造函数。
func NewOrderQueryService(repo domain.OrderRepository) *OrderQueryService {
	return &OrderQueryService{repo: repo}
}

func (s *OrderQueryService) GetOrder(ctx context.Context, orderID string) (*OrderDTO, error) {
	order, err := s.repo.Get(ctx, orderID)
	if err != nil || order == nil {
		return nil, err
	}
	return s.toDTO(order), nil
}

func (s *OrderQueryService) ListOrders(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	orders, total, err := s.repo.ListByUser(ctx, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*OrderDTO, 0, len(orders))
	for _, o := range orders {
		dtos = append(dtos, s.toDTO(o))
	}
	return dtos, total, nil
}

func (s *OrderQueryService) toDTO(o *domain.Order) *OrderDTO {
	return &OrderDTO{
		OrderID:        o.OrderID,
		UserID:         o.UserID,
		Symbol:         o.Symbol,
		Side:           string(o.Side),
		OrderType:      string(o.Type),
		Price:          decimal.NewFromFloat(o.Price).String(),
		Quantity:       decimal.NewFromFloat(o.Quantity).String(),
		FilledQuantity: decimal.NewFromFloat(o.FilledQuantity).String(),
		Status:         string(o.Status),
		TimeInForce:    string(o.TimeInForce),
		CreatedAt:      o.CreatedAt.Unix(),
		UpdatedAt:      o.UpdatedAt.Unix(),
	}
}
