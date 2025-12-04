package sender

import (
	"context"
	"fmt"

	"github.com/wyfcoding/financialTrading/internal/notification/domain"
)

// MockEmailSender 模拟邮件发送器
type MockEmailSender struct{}

// NewMockEmailSender 创建模拟邮件发送器
func NewMockEmailSender() domain.Sender {
	return &MockEmailSender{}
}

// Send 发送邮件
func (s *MockEmailSender) Send(ctx context.Context, target, subject, content string) error {
	fmt.Printf("[MockEmailSender] Sending email to %s: %s\n", target, subject)
	return nil
}

// MockSMSSender 模拟短信发送器
type MockSMSSender struct{}

// NewMockSMSSender 创建模拟短信发送器
func NewMockSMSSender() domain.Sender {
	return &MockSMSSender{}
}

// Send 发送短信
func (s *MockSMSSender) Send(ctx context.Context, target, subject, content string) error {
	fmt.Printf("[MockSMSSender] Sending SMS to %s: %s\n", target, content)
	return nil
}
