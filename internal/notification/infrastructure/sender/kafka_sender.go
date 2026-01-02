package sender

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
	"github.com/wyfcoding/pkg/messagequeue/kafka"
)

// KafkaNotificationSender 将通知指令发送到 Kafka，由专门的消费者服务（如阿里云 SMS / SendGrid 适配器）执行。
type KafkaNotificationSender struct {
	producer *kafka.Producer
	topic    string
}

// NotificationCommand 发送到 Kafka 的统一指令格式
type NotificationCommand struct {
	Target  string `json:"target"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

// NewKafkaNotificationSender 创建 Kafka 发送器
func NewKafkaNotificationSender(producer *kafka.Producer, topic string) domain.Sender {
	return &KafkaNotificationSender{
		producer: producer,
		topic:    topic,
	}
}

// Send 将通知推送到消息队列
func (s *KafkaNotificationSender) Send(ctx context.Context, target, subject, content string) error {
	cmd := NotificationCommand{
		Target:  target,
		Subject: subject,
		Content: content,
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal notification command: %w", err)
	}

	// 使用 Target 做 Key 保证同一接收者的时序性
	return s.producer.PublishToTopic(ctx, s.topic, []byte(target), payload)
}
