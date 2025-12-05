package sender

import (
	"context"

	"github.com/wyfcoding/financialTrading/internal/notification/domain"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// MockEmailSender 模拟邮件发送器
type MockEmailSender struct{}

// NewMockEmailSender 创建模拟邮件发送器
func NewMockEmailSender() domain.Sender {
	return &MockEmailSender{}
}

// Send 发送邮件（模拟实现）
func (s *MockEmailSender) Send(ctx context.Context, target, subject, content string) error {
	logger.Info(ctx, "Sending email notification",
		"sender", "MockEmailSender",
		"target", target,
		"subject", subject,
	)
	return nil
}

// MockSMSSender 模拟短信发送器
type MockSMSSender struct{}

// NewMockSMSSender 创建模拟短信发送器
func NewMockSMSSender() domain.Sender {
	return &MockSMSSender{}
}

// Send 发送短信（模拟实现）
func (s *MockSMSSender) Send(ctx context.Context, target, subject, content string) error {
	logger.Info(ctx, "Sending SMS notification",
		"sender", "MockSMSSender",
		"target", target,
		"content_length", len(content),
	)
	return nil
}
