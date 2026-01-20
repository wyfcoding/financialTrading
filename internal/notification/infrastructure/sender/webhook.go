package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

type WebhookSender struct {
	client *http.Client
}

func NewWebhookSender() domain.Sender {
	return &WebhookSender{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *WebhookSender) Send(ctx context.Context, target string, subject string, content string) error {
	slog.InfoContext(ctx, "sending webhook", "url", target, "subject", subject)

	payload := map[string]string{
		"text": fmt.Sprintf("*%s*\n%s", subject, content),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", target, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// resp, err := s.client.Do(req)
	// if err != nil { return err }
	// defer resp.Body.Close()

	slog.DebugContext(ctx, "Webhook triggered", "payload", string(body))
	return nil // 模拟环境中始终返回成功
}
