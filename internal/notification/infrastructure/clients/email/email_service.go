package email

import (
	"context"
	"fmt"
	"log/slog"

	notifModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/notification/domain/service"
)

type EmailService struct {
	sender  service.EmailSender
	baseURL string
	logger  *slog.Logger
}

func NewEmailService(sender service.EmailSender, baseURL string) *EmailService {
	return &EmailService{
		sender:  sender,
		baseURL: baseURL,
		logger:  slog.With(slog.String("component", "EmailService")),
	}
}

func (s *EmailService) SendConfirmation(ctx context.Context, to, repoName, token string) error {
	const op = "EmailService.SendConfirmation"

	msg := notifModel.EmailMessage{
		To:      to,
		Subject: fmt.Sprintf("Confirm subscription to %s", repoName),
		Body: fmt.Sprintf("Hello!\n\nTo confirm your subscription to %s, click here: %s/api/v1/confirm/%s",
			repoName, s.baseURL, token),
	}

	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	s.logger.DebugContext(ctx, "confirmation email sent",
		slog.String("op", op),
		slog.String("to", to),
		slog.String("repo", repoName),
	)
	return nil
}

func (s *EmailService) SendNotification(ctx context.Context, to, repoName, tag, token string) error {
	const op = "EmailService.SendNotification"

	msg := notifModel.EmailMessage{
		To:      to,
		Subject: fmt.Sprintf("New release: %s", repoName),
		Body: fmt.Sprintf(
			"Great news!\n\nA new version %s has been released for %s.\n\nUnsubscribe: %s/api/v1/unsubscribe/%s",
			tag,
			repoName,
			s.baseURL,
			token,
		),
	}
	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	s.logger.DebugContext(ctx, "release notification email sent",
		slog.String("op", op),
		slog.String("to", to),
		slog.String("repo", repoName),
		slog.String("tag", tag),
	)
	return nil
}
