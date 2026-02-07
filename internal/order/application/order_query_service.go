package application

import (
	"context"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
)

// OrderQueryService 处理所有订单相关的查询操作（Queries）。
type OrderQueryService struct {
	repo       domain.OrderRepository
	readRepo   domain.OrderReadRepository
	searchRepo domain.OrderSearchRepository
}

// NewOrderQueryService 构造函数。
func NewOrderQueryService(repo domain.OrderRepository, readRepo domain.OrderReadRepository, searchRepo domain.OrderSearchRepository) *OrderQueryService {
	return &OrderQueryService{
		repo:       repo,
		readRepo:   readRepo,
		searchRepo: searchRepo,
	}
}

func (s *OrderQueryService) GetOrder(ctx context.Context, orderID string) (*OrderDTO, error) {
	if s.readRepo != nil {
		if cached, err := s.readRepo.Get(ctx, orderID); err == nil && cached != nil {
			return toOrderDTO(cached), nil
		}
	}
	order, err := s.repo.Get(ctx, orderID)
	if err != nil || order == nil {
		return nil, err
	}
	if s.readRepo != nil {
		_ = s.readRepo.Save(ctx, order)
	}
	return toOrderDTO(order), nil
}

func (s *OrderQueryService) ListOrders(ctx context.Context, userID string, status domain.OrderStatus, limit, offset int) ([]*OrderDTO, int64, error) {
	orders, total, err := s.repo.ListByUser(ctx, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toOrderDTOs(orders), total, nil
}

func (s *OrderQueryService) SearchOrders(ctx context.Context, query map[string]any, limit int) ([]*OrderDTO, error) {
	if s.searchRepo == nil {
		return nil, nil
	}
	orders, err := s.searchRepo.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	return toOrderDTOs(orders), nil
}
