package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
)

type mockFields struct {
	subsRepo    *mocks.SubscriptionRepository
	userRepo    *mocks.UserRepository
	repoRepo    *mocks.RepositoryRepository
	ghClient    *mocks.GitHubClient
	emailSender *mocks.EmailSender
}

func newMockFields(t *testing.T) mockFields {
	t.Helper()
	return mockFields{
		subsRepo:    mocks.NewSubscriptionRepository(t),
		userRepo:    mocks.NewUserRepository(t),
		repoRepo:    mocks.NewRepositoryRepository(t),
		ghClient:    mocks.NewGitHubClient(t),
		emailSender: mocks.NewEmailSender(t),
	}
}

func newUC(f mockFields) *SubscriptionUseCase {
	return NewSubscriptionUseCase(
		f.subsRepo,
		f.ghClient,
		f.userRepo,
		f.repoRepo,
		f.emailSender,
	)
}

func TestSubscriptionUseCase_Subscribe(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		repoName  string
		setup     func(f mockFields, wg *sync.WaitGroup)
		wantErr   error
		expectErr bool
	}{
		{
			name:     "success - pending subscription created and confirmation email sent",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields, wg *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()
				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
				f.repoRepo.On("GetOrCreate", mock.Anything, "golang/go", "v1.22.0").
					Return(&model.Repository{ID: 10}, nil).Once()
				f.userRepo.On("GetOrCreate", mock.Anything, "user@test.com").
					Return(model.User{ID: 1}, nil).Once()
				f.subsRepo.On("CreatePending", mock.Anything, int64(1), int64(10), mock.AnythingOfType("string")).
					Return(int64(100), nil).Once()

				wg.Add(1)
				f.emailSender.On("SendConfirmation", mock.Anything, "user@test.com", "golang/go", mock.AnythingOfType("string")).
					Run(func(_ mock.Arguments) { wg.Done() }).
					Return(nil).
					Once()
			},
		},
		{
			name:     "error - repository does not exist on GitHub",
			email:    "user@test.com",
			repoName: "unknown/repo",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "unknown/repo").
					Return(false, nil).Once()
			},
			wantErr: service.ErrRepositoryNotFound,
		},
		{
			name:     "error - GitHub RepoExists call fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(false, errors.New("api connection error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - GetLatestRelease fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(nil, errors.New("github error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - repoRepo.GetOrCreate fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("GetOrCreate", mock.Anything, "golang/go", "v1.22.0").
					Return(nil, errors.New("repo storage error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - userRepo.GetOrCreate fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("GetOrCreate", mock.Anything, "golang/go", "v1.22.0").
					Return(&model.Repository{ID: 10}, nil).Once()

				f.userRepo.On("GetOrCreate", mock.Anything, "user@test.com").
					Return(model.User{}, errors.New("user repo error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - CreatePending fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.ghClient.On("RepoExists", mock.Anything, "golang/go").
					Return(true, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("GetOrCreate", mock.Anything, "golang/go", "v1.22.0").
					Return(&model.Repository{ID: 10}, nil).Once()

				f.userRepo.On("GetOrCreate", mock.Anything, "user@test.com").
					Return(model.User{ID: 1}, nil).Once()

				f.subsRepo.On("CreatePending", mock.Anything, int64(1), int64(10), mock.AnythingOfType("string")).
					Return(int64(0), errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFields(t)
			var wg sync.WaitGroup
			tt.setup(f, &wg)

			err := newUC(f).Subscribe(context.Background(), tt.email, tt.repoName)

			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			case tt.expectErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				done := make(chan struct{})
				go func() { wg.Wait(); close(done) }()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Fatal("timed out waiting for SendConfirmation goroutine")
				}
			}
		})
	}
}

func TestSubscriptionUseCase_Unsubscribe(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		repoName  string
		setup     func(f mockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:     "success - subscription deleted",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{ID: 1}, nil).Once()

				f.subsRepo.On("Delete", mock.Anything, int64(1), "golang/go").
					Return(nil).Once()
			},
		},
		{
			name:     "success - user not found, nothing to delete",
			email:    "unknown@test.com",
			repoName: "golang/go",
			setup: func(f mockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "unknown@test.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
			},
		},
		{
			name:     "error - GetByEmail fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Delete fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f mockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{ID: 1}, nil).Once()

				f.subsRepo.On("Delete", mock.Anything, int64(1), "golang/go").
					Return(errors.New("delete error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFields(t)
			tt.setup(f)

			err := newUC(f).Unsubscribe(context.Background(), tt.email, tt.repoName)

			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			case tt.expectErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				if tt.email == "unknown@test.com" {
					f.subsRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
				}
			}
		})
	}
}

func TestSubscriptionUseCase_ListByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		setup     func(f mockFields)
		expected  []model.Subscription
		expectErr bool
	}{
		{
			name:  "success - returns list",
			email: "user@test.com",
			setup: func(f mockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return([]model.Subscription{
						{ID: 1, RepositoryName: "golang/go", Confirmed: true},
						{ID: 2, RepositoryName: "google/uuid", Confirmed: false},
					}, nil).Once()
			},
			expected: []model.Subscription{
				{ID: 1, RepositoryName: "golang/go", Confirmed: true},
				{ID: 2, RepositoryName: "google/uuid", Confirmed: false},
			},
		},
		{
			name:  "success - empty list",
			email: "new@test.com",
			setup: func(f mockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "new@test.com").
					Return([]model.Subscription{}, nil).Once()
			},
			expected: []model.Subscription{},
		},
		{
			name:  "error - GetByEmail fails",
			email: "user@test.com",
			setup: func(f mockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFields(t)
			tt.setup(f)

			subs, err := newUC(f).ListByEmail(context.Background(), tt.email)

			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, subs)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, subs)
			f.userRepo.AssertNotCalled(t, "GetByEmail", mock.Anything, mock.Anything)
		})
	}
}

func TestSubscriptionUseCase_Confirm(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setup     func(f mockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:  "success",
			token: "valid-token",
			setup: func(f mockFields) {
				f.subsRepo.On("Confirm", mock.Anything, "valid-token").Return(nil).Once()
			},
		},
		{
			name:    "error - empty token, no repo call",
			token:   "",
			setup:   func(_ mockFields) {},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - invalid token from repo",
			token: "bad-token",
			setup: func(f mockFields) {
				f.subsRepo.On("Confirm", mock.Anything, "bad-token").
					Return(model.ErrInvalidToken).Once()
			},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - unexpected repo error",
			token: "some-token",
			setup: func(f mockFields) {
				f.subsRepo.On("Confirm", mock.Anything, "some-token").
					Return(errors.New("db failure")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFields(t)
			tt.setup(f)

			err := newUC(f).Confirm(context.Background(), tt.token)

			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			case tt.expectErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestSubscriptionUseCase_UnsubscribeByToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setup     func(f mockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:  "success",
			token: "valid-token",
			setup: func(f mockFields) {
				f.subsRepo.On("UnsubscribeByToken", mock.Anything, "valid-token").Return(nil).Once()
			},
		},
		{
			name:    "error - empty token",
			token:   "",
			setup:   func(_ mockFields) {},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - invalid token from repo",
			token: "expired",
			setup: func(f mockFields) {
				f.subsRepo.On("UnsubscribeByToken", mock.Anything, "expired").
					Return(model.ErrInvalidToken).Once()
			},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - unexpected repo error",
			token: "some-token",
			setup: func(f mockFields) {
				f.subsRepo.On("UnsubscribeByToken", mock.Anything, "some-token").
					Return(errors.New("db crash")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFields(t)
			tt.setup(f)

			err := newUC(f).UnsubscribeByToken(context.Background(), tt.token)

			switch {
			case tt.wantErr != nil:
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			case tt.expectErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestSubscriptionUseCase_ProcessNotifications(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f mockFields, wg *sync.WaitGroup)
		expectErr bool
	}{
		{
			name: "success - new release detected, notifications sent",
			setup: func(f mockFields, wg *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("UpdateLastSeenTag", mock.Anything, "golang/go", "v1.22.0").
					Return(nil).Once()

				subs := []model.Subscriber{
					{Email: "a@test.com", Token: "token-a"},
					{Email: "b@test.com", Token: "token-b"},
				}
				f.subsRepo.On("GetSubscribersByRepoID", mock.Anything, int64(1)).
					Return(subs, nil).Once()

				wg.Add(len(subs))
				for _, sub := range subs {
					f.emailSender.
						On("SendNotification", mock.Anything, sub.Email, "golang/go", "v1.22.0", sub.Token).
						Run(func(_ mock.Arguments) { wg.Done() }).
						Return(nil).Once()
				}
			},
		},
		{
			name: "success - tag unchanged, no notifications",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
			},
		},
		{
			name: "error - GetAll fails, returns error",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name: "partial - GitHub fails for one repo, continues",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(nil, errors.New("api error")).Once()
			},
		},
		{
			name: "partial - UpdateLastSeenTag fails, continues",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("UpdateLastSeenTag", mock.Anything, "golang/go", "v1.22.0").
					Return(errors.New("db error")).Once()
			},
		},
		{
			name: "partial - GetSubscribersByRepoID fails, continues",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("UpdateLastSeenTag", mock.Anything, "golang/go", "v1.22.0").
					Return(nil).Once()

				f.subsRepo.On("GetSubscribersByRepoID", mock.Anything, int64(1)).
					Return(nil, errors.New("db error")).Once()
			},
		},
		{
			name: "partial - SendNotification fails, continues silently",
			setup: func(f mockFields, wg *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

				f.repoRepo.On("UpdateLastSeenTag", mock.Anything, "golang/go", "v1.22.0").
					Return(nil).Once()

				f.subsRepo.On("GetSubscribersByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{{Email: "a@test.com", Token: "tok"}}, nil).Once()

				wg.Add(1)
				f.emailSender.
					On("SendNotification", mock.Anything, "a@test.com", "golang/go", "v1.22.0", "tok").
					Run(func(_ mock.Arguments) { wg.Done() }).
					Return(errors.New("SMTP error")).Once()
			},
		},
		{
			name: "success - nil release skipped",
			setup: func(f mockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).Return([]model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}, nil).Once()

				f.ghClient.On("GetLatestRelease", mock.Anything, "golang/go").
					Return(nil, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFields(t)
			var wg sync.WaitGroup
			tt.setup(f, &wg)

			err := newUC(f).ProcessNotifications(context.Background())

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			done := make(chan struct{})
			go func() { wg.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for email goroutines")
			}

			mock.AssertExpectationsForObjects(t,
				f.repoRepo, f.subsRepo, f.ghClient, f.emailSender,
			)
		})
	}
}
