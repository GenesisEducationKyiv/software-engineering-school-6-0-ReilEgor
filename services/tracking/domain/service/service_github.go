package service

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/domain/model"
)

var ErrReleaseNotFound = errors.New("no releases found for this repository")

//go:generate mockery --name GitHubClient --output ../../mocks --case underscore --outpkg mocks
type GitHubClient interface {
	RepoExists(ctx context.Context, fullName string) (bool, error)
	GetLatestRelease(ctx context.Context, fullName string) (*model.ReleaseInfo, error)
}
