//go:build settlement_elasticsearch
// +build settlement_elasticsearch

// Package elasticsearch 结算数据 ES 实现
package elasticsearch

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

type SettlementSearchRepository struct {
	client *elastic.Client
	index  string
}

func NewSettlementSearchRepository(client *elastic.Client) *SettlementSearchRepository {
	return &SettlementSearchRepository{
		client: client,
		index:  "settlement_records",
	}
}

// Save 将结算单投影到 ES
func (r *SettlementSearchRepository) Save(ctx context.Context, s *domain.Settlement) error {
	_, err := r.client.Index().
		Index(r.index).
		Id(fmt.Sprintf("%d", s.ID)).
		BodyJson(s).
		Do(ctx)
	return err
}

// Search 复杂查询示例
func (r *SettlementSearchRepository) Search(ctx context.Context, query map[string]interface{}) ([]*domain.Settlement, error) {
	// 实现基于多维条件的 ES 查询
	// 此处省略具体 DSL 构造逻辑
	return nil, nil
}

// AggregateByStatus 按状态统计结算金额
func (r *SettlementSearchRepository) AggregateByStatus(ctx context.Context) (map[string]float64, error) {
	// 使用 ES Aggregation 统计数据
	return nil, nil
}
