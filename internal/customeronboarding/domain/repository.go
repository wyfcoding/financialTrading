package domain

import (
	"context"
)

type OnboardingRepository interface {
	Save(ctx context.Context, app *OnboardingApplication) error
	Get(ctx context.Context, id string) (*OnboardingApplication, error)
	ListByEmail(ctx context.Context, email string) ([]*OnboardingApplication, error)
}
