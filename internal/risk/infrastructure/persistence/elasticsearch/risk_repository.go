package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/wyfcoding/financialtrading/internal/risk/domain"
	search_pkg "github.com/wyfcoding/pkg/search"
)

type riskSearchRepository struct {
	client          *search_pkg.Client
	assessmentIndex string
	alertIndex      string
}

// esSearchResponse ES 搜索响应结构

type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func NewRiskSearchRepository(client *search_pkg.Client, assessmentIndex, alertIndex string) domain.RiskSearchRepository {
	if client == nil {
		return nil
	}
	if assessmentIndex == "" {
		assessmentIndex = "risk_assessments"
	}
	if alertIndex == "" {
		alertIndex = "risk_alerts"
	}
	return &riskSearchRepository{
		client:          client,
		assessmentIndex: assessmentIndex,
		alertIndex:      alertIndex,
	}
}

func (r *riskSearchRepository) IndexAssessment(ctx context.Context, assessment *domain.RiskAssessment) error {
	if assessment == nil {
		return nil
	}
	id := assessment.ID
	return r.client.Index(ctx, r.assessmentIndex, id, assessment)
}

func (r *riskSearchRepository) IndexAlert(ctx context.Context, alert *domain.RiskAlert) error {
	if alert == nil {
		return nil
	}
	id := alert.ID
	return r.client.Index(ctx, r.alertIndex, id, alert)
}

func (r *riskSearchRepository) SearchAssessments(ctx context.Context, userID, symbol string, level domain.RiskLevel, limit, offset int) ([]*domain.RiskAssessment, int64, error) {
	query := buildAssessmentQuery(userID, symbol, level, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.assessmentIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	results := make([]*domain.RiskAssessment, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var a domain.RiskAssessment
		if err := json.Unmarshal(hit.Source, &a); err != nil {
			continue
		}
		results = append(results, &a)
	}
	return results, resp.Hits.Total.Value, nil
}

func (r *riskSearchRepository) SearchAlerts(ctx context.Context, userID, severity, alertType string, limit, offset int) ([]*domain.RiskAlert, int64, error) {
	query := buildAlertQuery(userID, severity, alertType, limit, offset)
	var resp esSearchResponse
	if err := r.client.Search(ctx, r.alertIndex, query, &resp); err != nil {
		return nil, 0, err
	}
	results := make([]*domain.RiskAlert, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var a domain.RiskAlert
		if err := json.Unmarshal(hit.Source, &a); err != nil {
			continue
		}
		results = append(results, &a)
	}
	return results, resp.Hits.Total.Value, nil
}

func buildAssessmentQuery(userID, symbol string, level domain.RiskLevel, limit, offset int) map[string]any {
	must := make([]map[string]any, 0, 3)
	if userID != "" {
		must = append(must, map[string]any{"term": map[string]any{"user_id": userID}})
	}
	if symbol != "" {
		must = append(must, map[string]any{"term": map[string]any{"symbol": symbol}})
	}
	if level != "" {
		must = append(must, map[string]any{"term": map[string]any{"risk_level": level}})
	}

	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"created_at": map[string]any{"order": "desc"}}},
	}
	if len(must) == 0 {
		query["query"] = map[string]any{"match_all": map[string]any{}}
		return query
	}
	query["query"] = map[string]any{
		"bool": map[string]any{
			"must": must,
		},
	}
	return query
}

func buildAlertQuery(userID, severity, alertType string, limit, offset int) map[string]any {
	must := make([]map[string]any, 0, 3)
	if userID != "" {
		must = append(must, map[string]any{"term": map[string]any{"user_id": userID}})
	}
	if severity != "" {
		must = append(must, map[string]any{"term": map[string]any{"severity": severity}})
	}
	if alertType != "" {
		must = append(must, map[string]any{"term": map[string]any{"alert_type": alertType}})
	}

	query := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []map[string]any{{"created_at": map[string]any{"order": "desc"}}},
	}
	if len(must) == 0 {
		query["query"] = map[string]any{"match_all": map[string]any{}}
		return query
	}
	query["query"] = map[string]any{
		"bool": map[string]any{
			"must": must,
		},
	}
	return query
}
