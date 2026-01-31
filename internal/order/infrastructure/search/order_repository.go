package search

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/search"
)

type orderSearchRepository struct {
	client *search.Client
	index  string
}

// NewOrderSearchRepository 创建订单搜索仓储实现。
func NewOrderSearchRepository(client *search.Client) domain.OrderSearchRepository {
	return &orderSearchRepository{
		client: client,
		index:  "orders",
	}
}

func (r *orderSearchRepository) Index(ctx context.Context, order *domain.Order) error {
	docID := fmt.Sprintf("%d", order.ID)
	return r.client.Index(ctx, r.index, docID, order)
}

func (r *orderSearchRepository) Search(ctx context.Context, query map[string]any, limit int) ([]*domain.Order, error) {
	// 复合查询支持：按价格范围、状态、交易对等进行过滤。
	esQuery := map[string]any{
		"query": query,
		"size":  limit,
	}

	var searchRes struct {
		Hits struct {
			Hits []struct {
				Source domain.Order `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := r.client.Search(ctx, r.index, esQuery, &searchRes); err != nil {
		return nil, fmt.Errorf("es search failed: %w", err)
	}

	orders := make([]*domain.Order, len(searchRes.Hits.Hits))
	for i, hit := range searchRes.Hits.Hits {
		o := hit.Source
		orders[i] = &o
	}

	return orders, nil
}
