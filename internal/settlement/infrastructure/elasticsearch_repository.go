//go:build settlement_elasticsearch
// +build settlement_elasticsearch

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

// ElasticsearchSettlementRepository Elasticsearch实现的结算仓储
type ElasticsearchSettlementRepository struct {
	client *elasticsearch.TypedClient
	index  string
}

// NewElasticsearchSettlementRepository 创建Elasticsearch结算仓储
func NewElasticsearchSettlementRepository(client *elasticsearch.TypedClient, index string) *ElasticsearchSettlementRepository {
	return &ElasticsearchSettlementRepository{
		client: client,
		index:  index,
	}
}

// SaveInstruction 保存结算指令
func (r *ElasticsearchSettlementRepository) SaveInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 序列化指令
	data, err := json.Marshal(instruction)
	if err != nil {
		return fmt.Errorf("failed to marshal instruction: %w", err)
	}

	// 保存到Elasticsearch
	_, err = r.client.Index(r.index).
		Id(instruction.ID).
		RequestJson(data).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to save instruction to Elasticsearch: %w", err)
	}

	return nil
}

// GetInstruction 获取结算指令
func (r *ElasticsearchSettlementRepository) GetInstruction(ctx context.Context, id string) (*domain.SettlementInstruction, error) {
	// 从Elasticsearch获取
	resp, err := r.client.Get(r.index, id).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instruction from Elasticsearch: %w", err)
	}

	// 反序列化
	var instruction domain.SettlementInstruction
	err = json.Unmarshal(resp.Source_, &instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal instruction: %w", err)
	}

	return &instruction, nil
}

// GetInstructionByRef 通过引用获取结算指令
func (r *ElasticsearchSettlementRepository) GetInstructionByRef(ctx context.Context, ref string) (*domain.SettlementInstruction, error) {
	// 构建查询
	query := &types.Query{
		Term: map[string]types.TermQuery{
			"instruction_ref": {Value: ref},
		},
	}

	// 执行查询
	resp, err := r.client.Search().
		Index(r.index).
		Request(&search.Request{
			Query: query,
			Size:  &[]int{1}[0],
		}).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to search instruction by ref: %w", err)
	}

	if len(resp.Hits.Hits) == 0 {
		return nil, fmt.Errorf("instruction not found for ref: %s", ref)
	}

	// 反序列化第一个结果
	var instruction domain.SettlementInstruction
	err = json.Unmarshal(resp.Hits.Hits[0].Source_, &instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal instruction: %w", err)
	}

	return &instruction, nil
}

// GetInstructionsByTrade 通过交易获取结算指令
func (r *ElasticsearchSettlementRepository) GetInstructionsByTrade(ctx context.Context, tradeID string) ([]*domain.SettlementInstruction, error) {
	// 构建查询
	query := &types.Query{
		Term: map[string]types.TermQuery{
			"trade_id": {Value: tradeID},
		},
	}

	// 执行查询
	resp, err := r.client.Search().
		Index(r.index).
		Request(&search.Request{
			Query: query,
			Size:  &[]int{100}[0],
		}).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to search instructions by trade: %w", err)
	}

	// 反序列化结果
	var instructions []*domain.SettlementInstruction
	for _, hit := range resp.Hits.Hits {
		var instruction domain.SettlementInstruction
		err = json.Unmarshal(hit.Source_, &instruction)
		if err != nil {
			fmt.Printf("Failed to unmarshal instruction: %v\n", err)
			continue
		}
		instructions = append(instructions, &instruction)
	}

	return instructions, nil
}

// GetInstructionsByBatch 通过批次获取结算指令
func (r *ElasticsearchSettlementRepository) GetInstructionsByBatch(ctx context.Context, batchID string) ([]*domain.SettlementInstruction, error) {
	// 构建查询
	query := &types.Query{
		Term: map[string]types.TermQuery{
			"batch_id": {Value: batchID},
		},
	}

	// 执行查询
	resp, err := r.client.Search().
		Index(r.index).
		Request(&search.Request{
			Query: query,
			Size:  &[]int{1000}[0],
		}).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to search instructions by batch: %w", err)
	}

	// 反序列化结果
	var instructions []*domain.SettlementInstruction
	for _, hit := range resp.Hits.Hits {
		var instruction domain.SettlementInstruction
		err = json.Unmarshal(hit.Source_, &instruction)
		if err != nil {
			fmt.Printf("Failed to unmarshal instruction: %v\n", err)
			continue
		}
		instructions = append(instructions, &instruction)
	}

	return instructions, nil
}

// GetPendingInstructionsByDate 获取指定日期的待结算指令
func (r *ElasticsearchSettlementRepository) GetPendingInstructionsByDate(ctx context.Context, settlementDate time.Time) ([]*domain.SettlementInstruction, error) {
	// 构建日期范围查询
	dateStr := settlementDate.Format("2006-01-02")

	query := &types.Query{
		Bool: &types.BoolQuery{
			Must: []types.Query{
				{
					Term: &types.TermQuery{
						Field: "settlement_date",
						Value: dateStr,
					},
				},
				{
					Term: &types.TermQuery{
						Field: "status",
						Value: string(domain.SettlementPending),
					},
				},
			},
		},
	}

	// 执行查询
	resp, err := r.client.Search().
		Index(r.index).
		Request(&search.Request{
			Query: query,
			Size:  &[]int{1000}[0],
		}).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to search pending instructions by date: %w", err)
	}

	// 反序列化结果
	var instructions []*domain.SettlementInstruction
	for _, hit := range resp.Hits.Hits {
		var instruction domain.SettlementInstruction
		err = json.Unmarshal(hit.Source_, &instruction)
		if err != nil {
			fmt.Printf("Failed to unmarshal instruction: %v\n", err)
			continue
		}
		instructions = append(instructions, &instruction)
	}

	return instructions, nil
}

// UpdateInstruction 更新结算指令
func (r *ElasticsearchSettlementRepository) UpdateInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 更新到Elasticsearch
	_, err := r.client.Update(r.index, instruction.ID).
		Doc(instruction).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to update instruction in Elasticsearch: %w", err)
	}

	return nil
}

// DeleteInstruction 删除结算指令
func (r *ElasticsearchSettlementRepository) DeleteInstruction(ctx context.Context, id string) error {
	// 从Elasticsearch删除
	_, err := r.client.Delete(r.index, id).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete instruction from Elasticsearch: %w", err)
	}

	return nil
}

// SearchInstructions 搜索结算指令
func (r *ElasticsearchSettlementRepository) SearchInstructions(ctx context.Context, criteria *domain.InstructionSearchCriteria) (*domain.InstructionSearchResult, error) {
	// 构建查询
	query := r.buildSearchQuery(criteria)

	// 执行查询
	resp, err := r.client.Search().
		Index(r.index).
		Request(&search.Request{
			Query: query,
			From:  &criteria.Offset,
			Size:  &criteria.Limit,
			Sort:  r.buildSort(criteria),
		}).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to search instructions: %w", err)
	}

	// 构建结果
	result := &domain.InstructionSearchResult{
		Total: int(resp.Hits.Total.Value),
		Page:  criteria.Offset/criteria.Limit + 1,
		Size:  criteria.Limit,
	}

	// 反序列化结果
	for _, hit := range resp.Hits.Hits {
		var instruction domain.SettlementInstruction
		err = json.Unmarshal(hit.Source_, &instruction)
		if err != nil {
			fmt.Printf("Failed to unmarshal instruction: %v\n", err)
			continue
		}
		result.Instructions = append(result.Instructions, &instruction)
	}

	return result, nil
}

// buildSearchQuery 构建搜索查询
func (r *ElasticsearchSettlementRepository) buildSearchQuery(criteria *domain.InstructionSearchCriteria) *types.Query {
	var mustQueries []types.Query

	// 状态过滤
	if criteria.Status != "" {
		mustQueries = append(mustQueries, types.Query{
			Term: &types.TermQuery{
				Field: "status",
				Value: criteria.Status,
			},
		})
	}

	// 符号过滤
	if criteria.Symbol != "" {
		mustQueries = append(mustQueries, types.Query{
			Term: &types.TermQuery{
				Field: "symbol",
				Value: criteria.Symbol,
			},
		})
	}

	// 日期范围过滤
	if !criteria.StartDate.IsZero() || !criteria.EndDate.IsZero() {
		dateRange := &types.DateRangeQuery{}

		if !criteria.StartDate.IsZero() {
			dateRange.Gte = &criteria.StartDate
		}

		if !criteria.EndDate.IsZero() {
			dateRange.Lte = &criteria.EndDate
		}

		mustQueries = append(mustQueries, types.Query{
			Range: map[string]types.RangeQuery{
				"settlement_date": dateRange,
			},
		})
	}

	// 金额范围过滤
	if criteria.MinAmount > 0 || criteria.MaxAmount > 0 {
		amountRange := &types.NumberRangeQuery{}

		if criteria.MinAmount > 0 {
			amountRange.Gte = &criteria.MinAmount
		}

		if criteria.MaxAmount > 0 {
			amountRange.Lte = &criteria.MaxAmount
		}

		mustQueries = append(mustQueries, types.Query{
			Range: map[string]types.RangeQuery{
				"amount": amountRange,
			},
		})
	}

	// 全文搜索
	if criteria.Keyword != "" {
		mustQueries = append(mustQueries, types.Query{
			MultiMatch: &types.MultiMatchQuery{
				Query:  criteria.Keyword,
				Fields: []string{"instruction_ref", "trade_id", "symbol", "buyer_id", "seller_id"},
			},
		})
	}

	if len(mustQueries) == 0 {
		return &types.Query{
			MatchAll: &types.MatchAllQuery{},
		}
	}

	return &types.Query{
		Bool: &types.BoolQuery{
			Must: mustQueries,
		},
	}
}

// buildSort 构建排序
func (r *ElasticsearchSettlementRepository) buildSort(criteria *domain.InstructionSearchCriteria) []types.Sort {
	var sorts []types.Sort

	// 默认按创建时间倒序
	if len(criteria.SortBy) == 0 {
		sorts = append(sorts, types.Sort{
			SortOptions: map[string]types.FieldSort{
				"created_at": {Order: &types.OrderDesc},
			},
		})
		return sorts
	}

	// 按指定字段排序
	for _, sortField := range criteria.SortBy {
		fieldSort := types.FieldSort{}

		if sortField.Direction == "asc" {
			fieldSort.Order = &types.OrderAsc
		} else {
			fieldSort.Order = &types.OrderDesc
		}

		sorts = append(sorts, types.Sort{
			SortOptions: map[string]types.FieldSort{
				sortField.Field: fieldSort,
			},
		})
	}

	return sorts
}

// AggregateInstructions 聚合结算指令
func (r *ElasticsearchSettlementRepository) AggregateInstructions(ctx context.Context, aggregation *domain.InstructionAggregation) (*domain.AggregationResult, error) {
	// 构建聚合查询
	aggs := r.buildAggregations(aggregation)

	// 执行查询
	resp, err := r.client.Search().
		Index(r.index).
		Request(&search.Request{
			Query: r.buildAggregationQuery(aggregation),
			Aggs:  aggs,
			Size:  &[]int{0}[0], // 只返回聚合结果
		}).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to aggregate instructions: %w", err)
	}

	// 解析聚合结果
	result := &domain.AggregationResult{
		Aggregations: make(map[string]interface{}),
	}

	// 解析每个聚合
	for name, agg := range resp.Aggregations {
		switch agg := agg.(type) {
		case *types.Buckets[string]:
			// 词项聚合
			buckets := make([]*domain.TermBucket, len(agg.Buckets))
			for i, bucket := range agg.Buckets {
				buckets[i] = &domain.TermBucket{
					Key:      bucket.Key,
					DocCount: int(bucket.DocCount),
				}
			}
			result.Aggregations[name] = buckets

		case *types.DateHistogramAggregate:
			// 日期直方图聚合
			buckets := make([]*domain.DateHistogramBucket, len(agg.Buckets))
			for i, bucket := range agg.Buckets {
				buckets[i] = &domain.DateHistogramBucket{
					Key:      bucket.KeyAsString,
					DocCount: int(bucket.DocCount),
				}
			}
			result.Aggregations[name] = buckets

		case *types.SumAggregate:
			// 求和聚合
			result.Aggregations[name] = agg.Value

		case *types.AvgAggregate:
			// 平均值聚合
			result.Aggregations[name] = agg.Value

		case *types.MinAggregate:
			// 最小值聚合
			result.Aggregations[name] = agg.Value

		case *types.MaxAggregate:
			// 最大值聚合
			result.Aggregations[name] = agg.Value
		}
	}

	return result, nil
}

// buildAggregationQuery 构建聚合查询
func (r *ElasticsearchSettlementRepository) buildAggregationQuery(aggregation *domain.InstructionAggregation) *types.Query {
	var mustQueries []types.Query

	// 日期范围过滤
	if !aggregation.StartDate.IsZero() || !aggregation.EndDate.IsZero() {
		dateRange := &types.DateRangeQuery{}

		if !aggregation.StartDate.IsZero() {
			dateRange.Gte = &aggregation.StartDate
		}

		if !aggregation.EndDate.IsZero() {
			dateRange.Lte = &aggregation.EndDate
		}

		mustQueries = append(mustQueries, types.Query{
			Range: map[string]types.RangeQuery{
				"settlement_date": dateRange,
			},
		})
	}

	// 状态过滤
	if aggregation.Status != "" {
		mustQueries = append(mustQueries, types.Query{
			Term: &types.TermQuery{
				Field: "status",
				Value: aggregation.Status,
			},
		})
	}

	if len(mustQueries) == 0 {
		return &types.Query{
			MatchAll: &types.MatchAllQuery{},
		}
	}

	return &types.Query{
		Bool: &types.BoolQuery{
			Must: mustQueries,
		},
	}
}

// buildAggregations 构建聚合
func (r *ElasticsearchSettlementRepository) buildAggregations(aggregation *domain.InstructionAggregation) map[string]types.Aggregations {
	aggs := make(map[string]types.Aggregations)

	// 添加每个聚合
	for _, agg := range aggregation.Aggregations {
		switch agg.Type {
		case "terms":
			// 词项聚合
			aggs[agg.Name] = types.Aggregations{
				Terms: &types.TermsAggregation{
					Field: &agg.Field,
					Size:  &agg.Size,
				},
			}

		case "date_histogram":
			// 日期直方图聚合
			aggs[agg.Name] = types.Aggregations{
				DateHistogram: &types.DateHistogramAggregation{
					Field:         &agg.Field,
					FixedInterval: &agg.Interval,
				},
			}

		case "sum":
			// 求和聚合
			aggs[agg.Name] = types.Aggregations{
				Sum: &types.SumAggregation{
					Field: &agg.Field,
				},
			}

		case "avg":
			// 平均值聚合
			aggs[agg.Name] = types.Aggregations{
				Avg: &types.AverageAggregation{
					Field: &agg.Field,
				},
			}

		case "min":
			// 最小值聚合
			aggs[agg.Name] = types.Aggregations{
				Min: &types.MinAggregation{
					Field: &agg.Field,
				},
			}

		case "max":
			// 最大值聚合
			aggs[agg.Name] = types.Aggregations{
				Max: &types.MaxAggregation{
					Field: &agg.Field,
				},
			}
		}
	}

	return aggs
}
