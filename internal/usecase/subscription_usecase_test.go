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

type subsMockFields struct {
	subsRepo     *mocks.SubscriptionRepository
	userRepo     *mocks.UserRepository
	repoRepo     *mocks.RepositoryRepository
	repoUC       *mocks.RepositoryUseCase
	emailService *mocks.EmailService
}

func newSubsMockFields(t *testing.T) subsMockFields {
	t.Helper()
	return subsMockFields{
		subsRepo:     mocks.NewSubscriptionRepository(t),
		userRepo:     mocks.NewUserRepository(t),
		repoRepo:     mocks.NewRepositoryRepository(t),
		repoUC:       mocks.NewRepositoryUseCase(t),
		emailService: mocks.NewEmailService(t),
	}
}

func newSubsUC(f subsMockFields) *SubscriptionUseCase {
	return NewSubscriptionUseCase(
		f.subsRepo,
		nil,
		f.repoUC,
		f.userRepo,
		f.emailService,
		f.repoRepo,
	)
}

func TestSubscriptionUseCase_Subscribe(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		repoName  string
		setup     func(f subsMockFields, wg *sync.WaitGroup)
		wantErr   error
		expectErr bool
	}{
		{
			name:     "success - pending subscription created (new user)",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields, wg *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 10, FullName: "golang/go"}, nil).Once()

				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
				f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
					Run(func(args mock.Arguments) {
						u, ok := args.Get(1).(*model.User)
						if !ok {
							return
						}
						u.ID = 5
					}).Return(nil).Once()

				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()

				wg.Add(1)
				f.emailService.On("SendConfirmation", mock.Anything, "user@test.com", "golang/go", mock.AnythingOfType("string")).
					Run(func(_ mock.Arguments) { wg.Done() }).
					Return(nil).
					Once()
			},
		},
		{
			name:     "success - pending subscription created (existing user)",
			email:    "existing@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields, wg *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 10, FullName: "golang/go"}, nil).Once()

				f.userRepo.On("GetByEmail", mock.Anything, "existing@test.com").
					Return(model.User{ID: 3}, nil).Once()

				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()

				wg.Add(1)
				f.emailService.On("SendConfirmation", mock.Anything, "existing@test.com", "golang/go", mock.AnythingOfType("string")).
					Run(func(_ mock.Arguments) { wg.Done() }).
					Return(nil).
					Once()
			},
		},
		{
			name:     "error - repository does not exist on GitHub",
			email:    "user@test.com",
			repoName: "unknown/repo",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "unknown/repo").
					Return((*model.Repository)(nil), service.ErrRepositoryNotFound).Once()
			},
			wantErr: service.ErrRepositoryNotFound,
		},
		{
			name:     "error - repoUC.GetOrCreate fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return((*model.Repository)(nil), errors.New("internal error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - userRepo.GetByEmail fails with unexpected error",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 10, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - userRepo.Create fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 10, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
				f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
					Return(errors.New("db write error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - subsRepo.Save fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 10, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{ID: 1}, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSubsMockFields(t)
			var wg sync.WaitGroup
			tt.setup(f, &wg)

			err := newSubsUC(f).Subscribe(context.Background(), tt.email, tt.repoName)

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
		setup     func(f subsMockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:     "success - subscription deleted",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields) {
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
			setup: func(f subsMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "unknown@test.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
			},
		},
		{
			name:     "error - GetByEmail fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(model.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Delete fails",
			email:    "user@test.com",
			repoName: "golang/go",
			setup: func(f subsMockFields) {
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
			f := newSubsMockFields(t)
			tt.setup(f)

			err := newSubsUC(f).Unsubscribe(context.Background(), tt.email, tt.repoName)

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
		setup     func(f subsMockFields)
		expected  []model.Subscription
		expectErr bool
	}{
		{
			name:  "success - returns list",
			email: "user@test.com",
			setup: func(f subsMockFields) {
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
			setup: func(f subsMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "new@test.com").
					Return([]model.Subscription{}, nil).Once()
			},
			expected: []model.Subscription{},
		},
		{
			name:  "error - GetByEmail fails",
			email: "user@test.com",
			setup: func(f subsMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "user@test.com").
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSubsMockFields(t)
			tt.setup(f)

			subs, err := newSubsUC(f).ListByEmail(context.Background(), tt.email)

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
		setup     func(f subsMockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:  "success",
			token: "valid-token",
			setup: func(f subsMockFields) {
				sub := &model.Subscription{ID: 1, Confirmed: false}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").Return(sub, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).Return(nil).Once()
			},
		},
		{
			name:    "error - empty token, no repo call",
			token:   "",
			setup:   func(_ subsMockFields) {},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - invalid token from GetByToken",
			token: "bad-token",
			setup: func(f subsMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "bad-token").
					Return((*model.Subscription)(nil), model.ErrInvalidToken).Once()
			},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - Save fails",
			token: "valid-token",
			setup: func(f subsMockFields) {
				sub := &model.Subscription{ID: 1, Confirmed: false}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").Return(sub, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(errors.New("db failure")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSubsMockFields(t)
			tt.setup(f)

			err := newSubsUC(f).Confirm(context.Background(), tt.token)

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
		setup     func(f subsMockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:  "success",
			token: "valid-token",
			setup: func(f subsMockFields) {
				sub := &model.Subscription{UserID: 1, RepositoryName: "golang/go"}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").Return(sub, nil).Once()
				f.subsRepo.On("Delete", mock.Anything, int64(1), "golang/go").Return(nil).Once()
			},
		},
		{
			name:    "error - empty token",
			token:   "",
			setup:   func(_ subsMockFields) {},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - invalid token from repo",
			token: "expired",
			setup: func(f subsMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "expired").
					Return((*model.Subscription)(nil), model.ErrInvalidToken).Once()
			},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - unexpected Delete error",
			token: "valid-token",
			setup: func(f subsMockFields) {
				sub := &model.Subscription{UserID: 1, RepositoryName: "golang/go"}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").Return(sub, nil).Once()
				f.subsRepo.On("Delete", mock.Anything, int64(1), "golang/go").
					Return(errors.New("db crash")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSubsMockFields(t)
			tt.setup(f)

			err := newSubsUC(f).UnsubscribeByToken(context.Background(), tt.token)

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
		setup     func(f subsMockFields, wg *sync.WaitGroup)
		expectErr bool
	}{
		{
			name: "success - new release detected, notifications sent",
			setup: func(f subsMockFields, wg *sync.WaitGroup) {
				repos := []model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}
				f.repoRepo.On("GetAll", mock.Anything).Return(repos, nil).Once()

				updatedRepo := &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}
				f.repoUC.On("CheckForUpdates", mock.Anything, repos[0]).
					Return(updatedRepo, nil).Once()

				subs := []model.Subscriber{
					{Email: "a@test.com", Token: "token-a"},
					{Email: "b@test.com", Token: "token-b"},
				}
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return(subs, nil).Once()

				wg.Add(len(subs))
				for _, sub := range subs {
					f.emailService.
						On("SendNotification", mock.Anything, sub.Email, "golang/go", "v1.22.0", sub.Token).
						Run(func(_ mock.Arguments) { wg.Done() }).
						Return(nil).Once()
				}
			},
		},
		{
			name: "success - tag unchanged, no notifications",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				repos := []model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
				}
				f.repoRepo.On("GetAll", mock.Anything).Return(repos, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repos[0]).
					Return((*model.Repository)(nil), nil).Once()
			},
		},
		{
			name: "error - GetAll fails",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name: "partial - CheckForUpdates fails, continues",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				repos := []model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}
				f.repoRepo.On("GetAll", mock.Anything).Return(repos, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repos[0]).
					Return((*model.Repository)(nil), errors.New("api error")).Once()
			},
		},
		{
			name: "partial - GetByRepoID fails, continues",
			setup: func(f subsMockFields, _ *sync.WaitGroup) {
				repos := []model.Repository{
					{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
				}
				f.repoRepo.On("GetAll", mock.Anything).Return(repos, nil).Once()

				updatedRepo := &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}
				f.repoUC.On("CheckForUpdates", mock.Anything, repos[0]).
					Return(updatedRepo, nil).Once()

				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return(nil, errors.New("db error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSubsMockFields(t)
			var wg sync.WaitGroup
			tt.setup(f, &wg)

			err := newSubsUC(f).ProcessNotifications(context.Background())

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
				f.repoRepo, f.subsRepo, f.repoUC, f.emailService,
			)
		})
	}
}
