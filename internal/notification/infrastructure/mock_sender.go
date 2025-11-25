package infrastructure

import (
	"context"
	"fmt"

	"github.com/fynnwu/FinancialTrading/internal/notification/domain"
)

// MockEmailSender 模拟邮件发送器
type MockEmailSender struct{}

func NewMockEmailSender() domain.Sender {
	return &MockEmailSender{}
}

func (s *MockEmailSender) Send(ctx context.Context, target string, subject string, content string) error {
	fmt.Printf("[MockEmailSender] Sending email to %s: Subject=%s\n", target, subject)
	return nil
}

// MockSMSSender 模拟短信发送器
type MockSMSSender struct{}

func NewMockSMSSender() domain.Sender {
	return &MockSMSSender{}
}

func (s *MockSMSSender) Send(ctx context.Context, target string, subject string, content string) error {
	fmt.Printf("[MockSMSSender] Sending SMS to %s: Content=%s\n", target, content)
	return nil
}
