package domain

import (
	"context"
	"time"
)

// Quote 行情快照
type Quote struct {
	Symbol    string    `json:"symbol"`
	BidPrice  float64   `json:"bid_price"`
	AskPrice  float64   `json:"ask_price"`
	LastPrice float64   `json:"last_price"`
	UpdatedAt time.Time `json:"updated_at"`
}

// QuoteRepository 读模型仓储（Redis）。
type QuoteRepository interface {
	Save(ctx context.Context, quote *Quote) error
	Get(ctx context.Context, symbol string) (*Quote, error)
	Delete(ctx context.Context, symbol string) error
}
