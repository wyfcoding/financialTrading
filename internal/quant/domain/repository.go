package domain

import "context"

type SignalRepository interface {
	Save(ctx context.Context, signal *Signal) error
	GetLatest(ctx context.Context, symbol string, indicator IndicatorType, period int) (*Signal, error)
}
