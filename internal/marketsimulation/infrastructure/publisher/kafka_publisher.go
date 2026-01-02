package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/marketsimulation/domain"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

// KafkaMarketDataPublisher 真实的 Kafka 市场数据发布者
type KafkaMarketDataPublisher struct {
	producer *kafka.Producer
	topic    string
}

// MarketDataEvent 定义发送到 Kafka 的行情消息格式
type MarketDataEvent struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Timestamp int64           `json:"timestamp"`
}

// NewKafkaMarketDataPublisher 创建 Kafka 发布者
func NewKafkaMarketDataPublisher(producer *kafka.Producer, topic string) domain.MarketDataPublisher {
	return &KafkaMarketDataPublisher{
		producer: producer,
		topic:    topic,
	}
}

// Publish 将市场数据推送到 Kafka
func (p *KafkaMarketDataPublisher) Publish(ctx context.Context, symbol string, price decimal.Decimal) error {
	event := MarketDataEvent{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now().UnixNano(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal market data: %w", err)
	}

	// 使用 Symbol 作为 Kafka Key 以保证同交易对顺序性
	return p.producer.PublishToTopic(ctx, p.topic, []byte(symbol), payload)
}
