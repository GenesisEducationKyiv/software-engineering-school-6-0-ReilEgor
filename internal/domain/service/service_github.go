package service

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

var (
	ErrRateLimitExceeded  = errors.New("github api rate limit exceeded")
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrReleaseNotFound    = errors.New("no releases found for this repository")
	ErrGitHubUnavailable  = errors.New("github service is temporarily unavailable")
)

//go:generate mockery --name GitHubClient --output ../../mocks --case underscore --outpkg mocks
type GitHubClient interface {
	RepoExists(ctx context.Context, fullName string) (bool, error)
	GetLatestRelease(ctx context.Context, fullName string) (*model.ReleaseInfo, error)
}
