package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
	"github.com/wyfcoding/pkg/search"
)

type auditESRepository struct {
	client *search.Client
	index  string
}

func NewAuditESRepository(client *search.Client) domain.AuditESRepository {
	return &auditESRepository{
		client: client,
		index:  "execution_audits",
	}
}

func (r *auditESRepository) Index(ctx context.Context, audit *domain.ExecutionAudit) error {
	return r.client.Index(ctx, r.index, audit.ID, audit)
}

func (r *auditESRepository) BatchIndex(ctx context.Context, audits []*domain.ExecutionAudit) error {
	if len(audits) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, a := range audits {
		meta := []byte(fmt.Sprintf(`{ "index" : { "_index" : "%s", "_id" : "%s" } }%s`, r.index, a.ID, "\n"))
		data, err := json.Marshal(a)
		if err != nil {
			return err
		}
		data = append(data, "\n"...)
		buf.Write(meta)
		buf.Write(data)
	}

	return r.client.Bulk(ctx, &buf)
}

func (r *auditESRepository) Search(ctx context.Context, query string, from, size int) ([]*domain.ExecutionAudit, int64, error) {
	searchQuery := map[string]any{
		"from": from,
		"size": size,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  query,
				"fields": []string{"user_id", "symbol", "venue", "algo_type", "trade_id", "order_id"},
			},
		},
		"sort": []map[string]any{
			{"timestamp": map[string]any{"order": "desc"}},
		},
	}

	var results struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source domain.ExecutionAudit `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := r.client.Search(ctx, r.index, searchQuery, &results); err != nil {
		return nil, 0, err
	}

	audits := make([]*domain.ExecutionAudit, 0, len(results.Hits.Hits))
	for i := range results.Hits.Hits {
		audits = append(audits, &results.Hits.Hits[i].Source)
	}

	return audits, results.Hits.Total.Value, nil
}
