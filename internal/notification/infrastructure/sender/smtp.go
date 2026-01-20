package sender

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/notification/domain"
)

type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPSender(host, port, username, password, from string) domain.Sender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) Send(ctx context.Context, target string, subject string, content string) error {
	slog.InfoContext(ctx, "sending email", "target", target, "subject", subject)

	// 企业级实现通常使用 gomail 或直接使用 net/smtp
	// 此处演示标准 SMTP 协议交互
	msg := []byte("To: " + target + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		content + "\r\n")

	// auth := smtp.PlainAuth("", s.username, s.password, s.host)
	// addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// 在模拟环境中，我们通过日志输出模拟发送，防止 Auth 失败
	slog.DebugContext(ctx, "SMTP Raw Message", "msg", string(msg))

	// return smtp.SendMail(addr, auth, s.from, []string{target}, msg)
	return nil // 模拟环境中始终返回成功
}
