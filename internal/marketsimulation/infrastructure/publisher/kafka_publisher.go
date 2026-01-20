package publisher

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

type KafkaMarketDataPublisher struct {
	producer *kafka.Producer
}

func NewKafkaMarketDataPublisher(producer *kafka.Producer) domain.MarketDataPublisher {
	return &KafkaMarketDataPublisher{producer: producer}
}

func (p *KafkaMarketDataPublisher) Publish(ctx context.Context, symbol string, price decimal.Decimal) error {
	msg := map[string]any{
		"symbol":    symbol,
		"price":     price.String(),
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(msg)

	return p.producer.PublishToTopic(ctx, "market.price", []byte(symbol), data)
}

// Note: Added time import below as it was used in map
