package mock

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

type MockEmailSender struct{}

func NewMockEmailSender() domain.Sender { return &MockEmailSender{} }

func (s *MockEmailSender) Send(ctx context.Context, to, subject, content string) error {
	slog.Info("Mock email sent", "to", to, "subject", subject)
	return nil
}

type MockSMSSender struct{}

func NewMockSMSSender() domain.Sender { return &MockSMSSender{} }

func (s *MockSMSSender) Send(ctx context.Context, to, subject, content string) error {
	slog.Info("Mock SMS sent", "to", to)
	return nil
}
