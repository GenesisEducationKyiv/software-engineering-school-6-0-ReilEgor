package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/ctxlog"

	notifModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/notification/internal/domain/service"
)

const componentEmailClient = "EmailClient"

type SMTPClient struct {
	host     string
	port     string
	from     string
	auth     smtp.Auth
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func NewSMTPClient(cfg config.EmailConfig) *SMTPClient {
	return &SMTPClient{
		host:     cfg.Host,
		port:     cfg.Port,
		from:     cfg.From,
		auth:     smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host),
		sendMail: smtp.SendMail,
	}
}

func (c *SMTPClient) log(ctx context.Context) *slog.Logger {
	return ctxlog.FromCtx(ctx).With(slog.String("component", componentEmailClient))
}

func (c *SMTPClient) Send(ctx context.Context, msg notifModel.EmailMessage) error {
	const op = "SMTPClient.Send"
	log := c.log(ctx).With(slog.String("op", op), slog.String("to", msg.To))
	log.DebugContext(ctx, "called")

	addr := fmt.Sprintf("%s:%s", c.host, c.port)

	rawMsg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		c.from, msg.To, msg.Subject, msg.Body,
	))

	if err := c.sendMail(addr, c.auth, c.from, []string{msg.To}, rawMsg); err != nil {
		log.ErrorContext(ctx, "failed to send email", slog.String("error", err.Error()))
		return classifySMTPError(err)
	}

	log.DebugContext(ctx, "email sent")
	return nil
}

func classifySMTPError(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "535") || strings.Contains(msg, "authentication failed") {
		return service.ErrAuthFailed
	}
	return service.ErrSMTPUnavailable
}
