package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
)

type userMockFields struct {
	subsRepo     *mocks.SubscriptionRepository
	userRepo     *mocks.UserRepository
	repoUC       *mocks.RepositoryUseCase
	emailService *mocks.EmailService
}

func newUserMockFields(t *testing.T) userMockFields {
	t.Helper()
	return userMockFields{
		subsRepo:     mocks.NewSubscriptionRepository(t),
		userRepo:     mocks.NewUserRepository(t),
		repoUC:       mocks.NewRepositoryUseCase(t),
		emailService: mocks.NewEmailService(t),
	}
}

func newUserUC(f userMockFields) *UserUseCase {
	return NewUserUseCase(f.subsRepo, f.userRepo, f.repoUC, f.emailService)
}

func TestUserUseCase_Subscribe(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		repoName  string
		setup     func(f userMockFields)
		expectErr bool
	}{
		{
			name:     "success - existing user subscribes to repo",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(model.User{ID: 10, Email: "user@example.com"}, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()
				f.emailService.On("SendConfirmation", mock.Anything, "user@example.com", "golang/go", mock.AnythingOfType("string")).
					Return(nil).
					Maybe()
			},
		},
		{
			name:     "success - new user created and subscribed",
			email:    "new@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "new@example.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
				f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
					Run(func(args mock.Arguments) {
						u, ok := args.Get(1).(*model.User)
						if !ok {
							return
						}
						u.ID = 99
					}).Return(nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()
				f.emailService.On("SendConfirmation", mock.Anything, "new@example.com", "golang/go", mock.AnythingOfType("string")).
					Return(nil).
					Maybe()
			},
		},
		{
			name:     "error - GetOrCreate fails",
			email:    "user@example.com",
			repoName: "unknown/repo",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "unknown/repo").
					Return((*model.Repository)(nil), errors.New("repo not found")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - GetByEmail unexpected DB error",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(model.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Create user fails",
			email:    "new@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "new@example.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
				f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
					Return(errors.New("db write error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Save subscription fails",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&model.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(model.User{ID: 10, Email: "user@example.com"}, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(errors.New("save error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newUserMockFields(t)
			tt.setup(f)

			err := newUserUC(f).Subscribe(context.Background(), tt.email, tt.repoName)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUserUseCase_Unsubscribe(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		repoName  string
		setup     func(f userMockFields)
		expectErr bool
	}{
		{
			name:     "success - user unsubscribed",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(model.User{ID: 10, Email: "user@example.com"}, nil).Once()
				f.subsRepo.On("Delete", mock.Anything, int64(10), "golang/go").
					Return(nil).Once()
			},
		},
		{
			name:     "success - user not found, nothing to unsubscribe",
			email:    "ghost@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "ghost@example.com").
					Return(model.User{}, model.ErrUserNotFound).Once()
			},
		},
		{
			name:     "error - GetByEmail unexpected DB error",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(model.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Delete subscription fails",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(model.User{ID: 10, Email: "user@example.com"}, nil).Once()
				f.subsRepo.On("Delete", mock.Anything, int64(10), "golang/go").
					Return(errors.New("delete error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newUserMockFields(t)
			tt.setup(f)

			err := newUserUC(f).Unsubscribe(context.Background(), tt.email, tt.repoName)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUserUseCase_ListByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		setup     func(f userMockFields)
		wantSubs  []model.Subscription
		expectErr bool
	}{
		{
			name:  "success - returns subscriptions",
			email: "user@example.com",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return([]model.Subscription{
						{UserID: 1, RepositoryName: "golang/go"},
						{UserID: 1, RepositoryName: "torvalds/linux"},
					}, nil).Once()
			},
			wantSubs: []model.Subscription{
				{UserID: 1, RepositoryName: "golang/go"},
				{UserID: 1, RepositoryName: "torvalds/linux"},
			},
		},
		{
			name:  "success - returns empty list",
			email: "empty@example.com",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "empty@example.com").
					Return([]model.Subscription{}, nil).Once()
			},
			wantSubs: []model.Subscription{},
		},
		{
			name:  "error - repo fails",
			email: "user@example.com",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newUserMockFields(t)
			tt.setup(f)

			got, err := newUserUC(f).ListByEmail(context.Background(), tt.email)

			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantSubs, got)
		})
	}
}

func TestUserUseCase_Confirm(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setup     func(f userMockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:  "success - subscription confirmed",
			token: "valid-token",
			setup: func(f userMockFields) {
				sub := &model.Subscription{
					UserID:         1,
					RepositoryName: "golang/go",
					Token:          "valid-token",
					Confirmed:      false,
				}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").
					Return(sub, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.MatchedBy(func(s *model.Subscription) bool {
					return s.Confirmed == true
				})).Return(nil).Once()
			},
		},
		{
			name:    "error - empty token",
			token:   "",
			setup:   func(_ userMockFields) {},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - token not found",
			token: "bad-token",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "bad-token").
					Return((*model.Subscription)(nil), model.ErrInvalidToken).Once()
			},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - Save fails after confirm",
			token: "valid-token",
			setup: func(f userMockFields) {
				sub := &model.Subscription{
					UserID:         1,
					RepositoryName: "golang/go",
					Token:          "valid-token",
					Confirmed:      false,
				}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").
					Return(sub, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(errors.New("db error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newUserMockFields(t)
			tt.setup(f)

			err := newUserUC(f).Confirm(context.Background(), tt.token)

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

func TestUserUseCase_UnsubscribeByToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setup     func(f userMockFields)
		wantErr   error
		expectErr bool
	}{
		{
			name:  "success - unsubscribed by token",
			token: "valid-token",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").
					Return(&model.Subscription{UserID: 5, RepositoryName: "golang/go", Token: "valid-token"}, nil).
					Once()
				f.subsRepo.On("Delete", mock.Anything, int64(5), "golang/go").
					Return(nil).Once()
			},
		},
		{
			name:    "error - empty token",
			token:   "",
			setup:   func(_ userMockFields) {},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - token not found",
			token: "bad-token",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "bad-token").
					Return((*model.Subscription)(nil), model.ErrInvalidToken).Once()
			},
			wantErr: model.ErrInvalidToken,
		},
		{
			name:  "error - Delete fails",
			token: "valid-token",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").
					Return(&model.Subscription{UserID: 5, RepositoryName: "golang/go", Token: "valid-token"}, nil).
					Once()
				f.subsRepo.On("Delete", mock.Anything, int64(5), "golang/go").
					Return(errors.New("delete error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newUserMockFields(t)
			tt.setup(f)

			err := newUserUC(f).UnsubscribeByToken(context.Background(), tt.token)

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
