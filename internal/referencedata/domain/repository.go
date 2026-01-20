package domain

import "context"

type ReferenceRepository interface {
	Save(ctx context.Context, instrument *Instrument) error
	GetInstrument(ctx context.Context, symbol string) (*Instrument, error)
	ListInstruments(ctx context.Context) ([]*Instrument, error)
}
