package service

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

var (
	ErrSMTPUnavailable = errors.New("email server is unreachable")
	ErrAuthFailed      = errors.New("email service authentication failed")
)

//go:generate mockery --name EmailSender --output ../../mocks --case underscore --outpkg mocks
type EmailSender interface {
	Send(ctx context.Context, msg model.EmailMessage) error
}

//go:generate mockery --name EmailService --output ../../mocks --case underscore --outpkg mocks
type EmailService interface {
	SendConfirmation(ctx context.Context, to, repoName, token string) error
	SendNotification(ctx context.Context, to, repoName, tagName, token string) error
}
