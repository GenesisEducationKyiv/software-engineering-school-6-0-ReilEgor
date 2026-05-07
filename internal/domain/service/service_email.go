package service

import (
	"context"
	"errors"
)

var (
	ErrSMTPUnavailable = errors.New("email server is unreachable")
	ErrAuthFailed      = errors.New("email service authentication failed")
)

//go:generate mockery --name EmailSender --output ../../mocks --case underscore --outpkg mocks
type EmailSender interface {
	SendNotification(ctx context.Context, to, repoName, tagName, token string) error
	SendConfirmation(ctx context.Context, to, repoName, token string) error
}
