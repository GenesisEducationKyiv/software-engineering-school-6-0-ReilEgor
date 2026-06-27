package service

import "context"

//go:generate mockery --name ConfirmationSender --output ../../../shared/mocks --case underscore --outpkg mocks
type ConfirmationSender interface {
	SendConfirmation(ctx context.Context, to, repoName, token string) error
}
