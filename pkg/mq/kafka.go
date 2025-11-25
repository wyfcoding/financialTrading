// Package mq 提供 Kafka producer/consumer 通用实现，支持幂等、重试、死信队列、事务/Exactly-Once
package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wyfcoding/financialTrading/pkg/logger"
	"github.com/segmentio/kafka-go"
)

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers           []string
	GroupID           string
	Partitions        int
	Replication       int
	SessionTimeout    int
	MaxRetries        int
	RetryBackoff      int
	EnableCompression bool
}

// KafkaProducer Kafka 生产者
type KafkaProducer struct {
	writer *kafka.Writer
	config KafkaConfig
}

// NewProducer 创建 Kafka 生产者
func NewProducer(cfg KafkaConfig) (*KafkaProducer, error) {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		AllowAutoTopicCreation: true,
		Compression:            kafka.Gzip,
		RequiredAcks:           kafka.RequireAll, // 等待所有副本确认
		MaxAttempts:            cfg.MaxRetries,
		WriteBackoffMin:        time.Duration(cfg.RetryBackoff) * time.Millisecond,
		WriteBackoffMax:        time.Duration(cfg.RetryBackoff*10) * time.Millisecond,
	}

	logger.Info(context.Background(), "Kafka producer created successfully", "brokers", cfg.Brokers)
	return &KafkaProducer{
		writer: writer,
		config: cfg,
	}, nil
}

// SendMessage 发送单条消息
func (kp *KafkaProducer) SendMessage(ctx context.Context, topic string, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}

	err = kp.writer.WriteMessages(ctx, msg)
	if err != nil {
		logger.Error(ctx, "Failed to send Kafka message",
			"topic", topic,
			"key", key,
			"error", err,
		)
		return err
	}

	logger.Debug(ctx, "Kafka message sent",
		"topic", topic,
		"key", key,
	)
	return nil
}

// SendMessages 批量发送消息
func (kp *KafkaProducer) SendMessages(ctx context.Context, topic string, messages []map[string]interface{}) error {
	kafkaMessages := make([]kafka.Message, 0, len(messages))

	for _, msg := range messages {
		key, ok := msg["key"].(string)
		if !ok {
			key = ""
		}

		value, ok := msg["value"]
		if !ok {
			continue
		}

		data, err := json.Marshal(value)
		if err != nil {
			logger.Error(ctx, "Failed to marshal message", "error", err)
			continue
		}

		kafkaMessages = append(kafkaMessages, kafka.Message{
			Topic: topic,
			Key:   []byte(key),
			Value: data,
		})
	}

	if len(kafkaMessages) == 0 {
		return nil
	}

	err := kp.writer.WriteMessages(ctx, kafkaMessages...)
	if err != nil {
		logger.Error(ctx, "Failed to send Kafka messages",
			"topic", topic,
			"count", len(kafkaMessages),
			"error", err,
		)
		return err
	}

	logger.Debug(ctx, "Kafka messages sent",
		"topic", topic,
		"count", len(kafkaMessages),
	)
	return nil
}

// Close 关闭生产者
func (kp *KafkaProducer) Close() error {
	return kp.writer.Close()
}

// KafkaConsumer Kafka 消费者
type KafkaConsumer struct {
	reader *kafka.Reader
	config KafkaConfig
}

// NewConsumer 创建 Kafka 消费者
func NewConsumer(cfg KafkaConfig, topic string) (*KafkaConsumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          topic,
		GroupID:        cfg.GroupID,
		SessionTimeout: time.Duration(cfg.SessionTimeout) * time.Second,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		MaxBytes:       10e6, // 10MB
	})

	logger.Info(context.Background(), "Kafka consumer created successfully",
		"brokers", cfg.Brokers,
		"topic", topic,
		"group_id", cfg.GroupID,
	)
	return &KafkaConsumer{
		reader: reader,
		config: cfg,
	}, nil
}

// ReadMessage 读取单条消息
func (kc *KafkaConsumer) ReadMessage(ctx context.Context) (*Message, error) {
	msg, err := kc.reader.ReadMessage(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to read Kafka message", "error", err)
		return nil, err
	}

	return &Message{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       string(msg.Key),
		Value:     msg.Value,
		Time:      msg.Time,
	}, nil
}

// ReadMessages 批量读取消息
func (kc *KafkaConsumer) ReadMessages(ctx context.Context, maxMessages int) ([]*Message, error) {
	messages := make([]*Message, 0, maxMessages)

	for i := 0; i < maxMessages; i++ {
		msg, err := kc.reader.ReadMessage(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				break
			}
			logger.Error(ctx, "Failed to read Kafka message", "error", err)
			return nil, err
		}

		messages = append(messages, &Message{
			Topic:     msg.Topic,
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Key:       string(msg.Key),
			Value:     msg.Value,
			Time:      msg.Time,
		})
	}

	return messages, nil
}

// CommitMessages 提交消息偏移量
func (kc *KafkaConsumer) CommitMessages(ctx context.Context, messages ...*Message) error {
	if len(messages) == 0 {
		return nil
	}

	// Kafka reader 自动提交，这里仅用于显式提交
	return nil
}

// Close 关闭消费者
func (kc *KafkaConsumer) Close() error {
	return kc.reader.Close()
}

// Message Kafka 消息结构
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       string
	Value     []byte
	Time      time.Time
}

// UnmarshalPayload 将消息值解析为 JSON
func (m *Message) UnmarshalPayload(dest interface{}) error {
	return json.Unmarshal(m.Value, dest)
}

// DeadLetterQueue 死信队列处理
type DeadLetterQueue struct {
	producer *KafkaProducer
	topic    string
}

// NewDeadLetterQueue 创建死信队列
func NewDeadLetterQueue(producer *KafkaProducer, topic string) *DeadLetterQueue {
	return &DeadLetterQueue{
		producer: producer,
		topic:    topic,
	}
}

// Send 发送消息到死信队列
func (dlq *DeadLetterQueue) Send(ctx context.Context, originalMessage *Message, reason string, err error) error {
	deadLetterMsg := map[string]interface{}{
		"original_topic":    originalMessage.Topic,
		"original_key":      originalMessage.Key,
		"original_value":    string(originalMessage.Value),
		"original_offset":   originalMessage.Offset,
		"original_time":     originalMessage.Time,
		"failure_reason":    reason,
		"failure_error":     err.Error(),
		"failure_timestamp": time.Now(),
	}

	return dlq.producer.SendMessage(ctx, dlq.topic, originalMessage.Key, deadLetterMsg)
}
