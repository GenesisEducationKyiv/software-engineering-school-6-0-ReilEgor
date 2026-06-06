package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/shared/domain/service"
)

const componentEmailClient = "EmailClient"

type SMTPClient struct {
	host     string
	port     string
	from     string
	auth     smtp.Auth
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
	logger   *slog.Logger
}

func NewSMTPClient(cfg config.EmailConfig) *SMTPClient {
	return &SMTPClient{
		host:     cfg.Host,
		port:     cfg.Port,
		from:     cfg.From,
		auth:     smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host),
		logger:   slog.With(slog.String("component", componentEmailClient)),
		sendMail: smtp.SendMail,
	}
}

func (c *SMTPClient) Send(ctx context.Context, msg model.EmailMessage) error {
	addr := fmt.Sprintf("%s:%s", c.host, c.port)

	rawMsg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		c.from, msg.To, msg.Subject, msg.Body,
	))

	if err := c.sendMail(addr, c.auth, c.from, []string{msg.To}, rawMsg); err != nil {
		c.logger.ErrorContext(ctx, "failed to send email", slog.String("to", msg.To), slog.Any("error", err))
		return classifySMTPError(err)
	}
	return nil
}

func classifySMTPError(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "535") || strings.Contains(msg, "authentication failed") {
		return service.ErrAuthFailed
	}
	return service.ErrSMTPUnavailable
}
