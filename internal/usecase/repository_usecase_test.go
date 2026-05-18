package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
)

type repoMockFields struct {
	repoRepo *mocks.RepositoryRepository
	ghClient *mocks.GitHubClient
}

func newRepoMockFields(t *testing.T) repoMockFields {
	t.Helper()
	return repoMockFields{
		repoRepo: mocks.NewRepositoryRepository(t),
		ghClient: mocks.NewGitHubClient(t),
	}
}

func newRepoUC(f repoMockFields) *RepositoryUseCase {
	return NewRepositoryUseCase(f.repoRepo, f.ghClient)
}

func TestRepositoryUseCase_GetOrCreate(t *testing.T) {
	tests := []struct {
		name      string
		repoName  string
		setup     func(f repoMockFields)
		wantRepo  *model.Repository
		wantErr   error
		expectErr bool
	}{
		{
			name:     "success - repo already exists in DB",
			repoName: "golang/go",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).Once()
			},
			wantRepo: &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
		},
		{
			name:     "success - repo not in DB, created from GitHub",
			repoName: "golang/go",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), model.ErrRepositoryNotFound).Once()
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
				f.repoRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Repository")).
					Run(func(args mock.Arguments) {
						repo, ok := args.Get(1).(*model.Repository)
						if !ok {
							return
						}
						repo.ID = 10
					}).Return(nil).Once()
			},
			wantRepo: &model.Repository{ID: 10, FullName: "golang/go", LastSeenTag: "v1.22.0"},
		},
		{
			name:     "error - repo not found on GitHub",
			repoName: "unknown/repo",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "unknown/repo").
					Return((*model.Repository)(nil), model.ErrRepositoryNotFound).Once()
				f.ghClient.On("RepoExists", mock.Anything, "unknown/repo").
					Return(false, nil).Once()
			},
			wantErr: service.ErrRepositoryNotFound,
		},
		{
			name:     "error - GetByName unexpected DB error",
			repoName: "golang/go",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - RepoExists GitHub call fails",
			repoName: "golang/go",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), model.ErrRepositoryNotFound).Once()
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(false, errors.New("api error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - GetLatestRelease fails",
			repoName: "golang/go",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), model.ErrRepositoryNotFound).Once()
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(nil, errors.New("github error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Create in DB fails",
			repoName: "golang/go",
			setup: func(f repoMockFields) {
				f.repoRepo.On("GetByName", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), model.ErrRepositoryNotFound).Once()
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
				f.repoRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Repository")).
					Return(errors.New("db write error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRepoMockFields(t)
			tt.setup(f)

			got, err := newRepoUC(f).GetOrCreate(context.Background(), tt.repoName)

			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			case tt.expectErr:
				require.Error(t, err)
				assert.Nil(t, got)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.wantRepo, got)
			}
		})
	}
}

func TestRepositoryUseCase_CheckForUpdates(t *testing.T) {
	tests := []struct {
		name      string
		repo      model.Repository
		setup     func(f repoMockFields)
		wantRepo  *model.Repository
		expectErr bool
	}{
		{
			name: "success - context has deadline (10s timeout)",
			repo: model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
			setup: func(f repoMockFields) {
				f.ghClient.On("GetLatestRelease",
					mock.MatchedBy(func(ctx context.Context) bool {
						deadline, hasDeadline := ctx.Deadline()
						if !hasDeadline {
							return false
						}
						remaining := time.Until(deadline)
						return remaining > (checkForUpdatesCtxTimeout-1)*time.Second &&
							remaining <= checkForUpdatesCtxTimeout*time.Second
					}),
					"golang/go",
				).Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
				f.repoRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Repository")).
					Return(nil).Once()
			},
			wantRepo: &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
		},
		{
			name: "success - new release detected, repo updated",
			repo: model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
			setup: func(f repoMockFields) {
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
				f.repoRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Repository")).
					Return(nil).Once()
			},
			wantRepo: &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
		},
		{
			name: "success - tag unchanged, returns nil",
			repo: model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
			setup: func(f repoMockFields) {
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
			},
			wantRepo: nil,
		},
		{
			name: "success - nil release, returns nil",
			repo: model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
			setup: func(f repoMockFields) {
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(nil, nil).Once()
			},
			wantRepo: nil,
		},
		{
			name: "error - GetLatestRelease fails",
			repo: model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
			setup: func(f repoMockFields) {
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(nil, errors.New("api error")).Once()
			},
			expectErr: true,
		},
		{
			name: "error - Update in DB fails",
			repo: model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
			setup: func(f repoMockFields) {
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
				f.repoRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Repository")).
					Return(errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRepoMockFields(t)
			tt.setup(f)

			got, err := newRepoUC(f).CheckForUpdates(context.Background(), tt.repo)

			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRepo, got)
		})
	}
}
