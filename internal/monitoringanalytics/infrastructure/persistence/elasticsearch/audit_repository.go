package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/wyfcoding/financialtrading/internal/monitoringanalytics/domain"
)

type auditESRepository struct {
	client *elasticsearch.Client
	index  string
}

func NewAuditESRepository(client *elasticsearch.Client) domain.AuditESRepository {
	return &auditESRepository{
		client: client,
		index:  "execution_audits",
	}
}

func (r *auditESRepository) Index(ctx context.Context, audit *domain.ExecutionAudit) error {
	data, err := json.Marshal(audit)
	if err != nil {
		return err
	}

	res, err := r.client.Index(
		r.index,
		bytes.NewReader(data),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithDocumentID(audit.ID),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index audit: %s", res.String())
	}
	return nil
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

	res, err := r.client.Bulk(bytes.NewReader(buf.Bytes()), r.client.Bulk.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to bulk index audits: %s", res.String())
	}
	return nil
}

func (r *auditESRepository) Search(ctx context.Context, query string, from, size int) ([]*domain.ExecutionAudit, int64, error) {
	var buf bytes.Buffer
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

	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		return nil, 0, err
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.index),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("search failed: %s", res.String())
	}

	var rMap map[string]any
	if err := json.NewDecoder(res.Body).Decode(&rMap); err != nil {
		return nil, 0, err
	}

	hits := rMap["hits"].(map[string]any)
	total := int64(hits["total"].(map[string]any)["value"].(float64))

	results := make([]*domain.ExecutionAudit, 0)
	for _, hit := range hits["hits"].([]any) {
		source := hit.(map[string]any)["_source"]
		data, _ := json.Marshal(source)
		var a domain.ExecutionAudit
		_ = json.Unmarshal(data, &a)
		results = append(results, &a)
	}

	return results, total, nil
}
