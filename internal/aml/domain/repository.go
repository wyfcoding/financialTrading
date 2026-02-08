package domain

import (
	"context"
)

type AMLRepository interface {
	SaveAlert(ctx context.Context, alert *AMLAlert) error
	GetAlert(ctx context.Context, id string) (*AMLAlert, error)
	ListAlertsByStatus(ctx context.Context, status string) ([]*AMLAlert, error)

	SaveRiskScore(ctx context.Context, score *UserRiskScore) error
	GetRiskScore(ctx context.Context, userID string) (*UserRiskScore, error)
}
