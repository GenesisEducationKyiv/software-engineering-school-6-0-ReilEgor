package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/ReilEgor/RepoNotifier/internal/config"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
)

const (
	componentEmailClient = "EmailClient"

	emailSubjectTemplate = "New release in %s!"
	emailBodyTemplate    = "Hello!\n\nA new version %s has been released for %s.\n" +
		"Check it out here: https://github.com/%s/releases/tag/%s\n\n" +
		"---\n" +
		"Unsubscribe: %s/api/v1/unsubscribe/%s\n\n" +
		"Best regards,\nRepoNotifier"

	emailMsgTemplate = "From: %s\r\n" +
		"To: %s\r\n" +
		"Subject: %s\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"%s\r\n"

	confirmSubjectTemplate = "Confirm your subscription to %s"
	confirmBodyTemplate    = "Hello!\n\nTo start receiving notifications for %s, " +
		"please confirm your subscription:\n\n" +
		"%s/api/v1/confirm/%s\n\n" +
		"Best regards,\nRepoNotifier"
)

const (
	errMsgSendMail = "failed to send email"
	// errMsgBuildMsg = "failed to build message".
)

func (c *SMTPClient) buildMessage(to, repoName, tagName, token string) []byte {
	subject := fmt.Sprintf(emailSubjectTemplate, repoName)
	body := fmt.Sprintf(emailBodyTemplate, tagName, repoName, repoName, tagName, c.baseURL, token)
	return []byte(fmt.Sprintf(emailMsgTemplate, c.from, to, subject, body))
}

func (c *SMTPClient) buildConfirmMessage(to, repoName, token string) []byte {
	subject := fmt.Sprintf(confirmSubjectTemplate, repoName)
	body := fmt.Sprintf(confirmBodyTemplate, repoName, c.baseURL, token)
	return []byte(fmt.Sprintf(emailMsgTemplate, c.from, to, subject, body))
}

type SMTPClient struct {
	host     config.EmailHostType
	port     config.EmailPortType
	from     config.EmailFromType
	auth     smtp.Auth
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
	logger   *slog.Logger
	baseURL  config.AppBaseURLType
}

func NewSMTPClient(
	host config.EmailHostType,
	port config.EmailPortType,
	from config.EmailFromType,
	password config.EmailPasswordType,
	user config.EmailUserType,
	baseURL config.AppBaseURLType,
) *SMTPClient {
	return &SMTPClient{
		host:     host,
		port:     port,
		from:     from,
		auth:     smtp.PlainAuth("", string(user), string(password), string(host)),
		logger:   slog.With(slog.String("component", componentEmailClient)),
		sendMail: smtp.SendMail,
		baseURL:  baseURL,
	}
}

func classifySMTPError(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "535") || strings.Contains(msg, "authentication failed") {
		return service.ErrAuthFailed
	}
	return service.ErrSMTPUnavailable
}

func (c *SMTPClient) sendEmail(ctx context.Context, op, to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%s", c.host, c.port)
	if err := c.sendMail(addr, c.auth, string(c.from), []string{to}, msg); err != nil {
		c.logger.ErrorContext(ctx, errMsgSendMail,
			slog.String("op", op),
			slog.String("to", to),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: %w", op, classifySMTPError(err))
	}
	return nil
}

func (c *SMTPClient) SendNotification(ctx context.Context, to, repoName, tagName, token string) error {
	const op = "SMTPClient.SendNotification"
	msg := c.buildMessage(to, repoName, tagName, token)
	if err := c.sendEmail(ctx, op, to, msg); err != nil {
		return err
	}
	c.logger.InfoContext(ctx, "notification email sent",
		slog.String("to", to),
		slog.String("repo", repoName),
		slog.String("tag", tagName),
	)
	return nil
}

func (c *SMTPClient) SendConfirmation(ctx context.Context, to, repoName, token string) error {
	const op = "SMTPClient.SendConfirmation"
	msg := c.buildConfirmMessage(to, repoName, token)
	if err := c.sendEmail(ctx, op, to, msg); err != nil {
		return err
	}
	c.logger.InfoContext(ctx, "confirmation email sent", slog.String("to", to))
	return nil
}
