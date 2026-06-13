package service

import "context"

type SendNotificationCommand struct {
	Email    string `json:"email"`
	RepoName string `json:"repo_name"`
	Tag      string `json:"tag"`
	Token    string `json:"token"`
}

//go:generate mockery --name MessagePublisher --output ../../mocks --case underscore --outpkg mocks
type MessagePublisher interface {
	Publish(ctx context.Context, cmd SendNotificationCommand) error
}