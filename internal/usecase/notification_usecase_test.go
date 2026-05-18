package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
)

type notifMockFields struct {
	subsRepo     *mocks.SubscriptionRepository
	repoRepo     *mocks.RepositoryRepository
	repoUC       *mocks.RepositoryUseCase
	emailService *mocks.EmailService
}

func newNotifMockFields(t *testing.T) notifMockFields {
	t.Helper()
	return notifMockFields{
		subsRepo:     mocks.NewSubscriptionRepository(t),
		repoRepo:     mocks.NewRepositoryRepository(t),
		repoUC:       mocks.NewRepositoryUseCase(t),
		emailService: mocks.NewEmailService(t),
	}
}

func newNotifUC(f notifMockFields) *NotificationUseCase {
	return NewNotificationUseCase(f.subsRepo, f.repoRepo, f.repoUC, f.emailService)
}

func TestNotificationUseCase_ProcessNotifications(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f notifMockFields)
		expectErr bool
	}{
		{
			name: "success - no repos",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{}, nil).Once()
			},
		},
		{
			name: "success - repo has no new release",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}).
					Return((*model.Repository)(nil), nil).
					Once()
			},
		},
		{
			name: "success - new release, notifications sent",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).
					Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{
						{Email: "alice@example.com", Token: "tok-a"},
						{Email: "bob@example.com", Token: "tok-b"},
					}, nil).Once()
				f.emailService.On("SendNotification", mock.Anything, "alice@example.com", "golang/go", "v1.22.0", "tok-a").
					Return(nil).
					Once()
				f.emailService.On("SendNotification", mock.Anything, "bob@example.com", "golang/go", "v1.22.0", "tok-b").
					Return(nil).
					Once()
			},
		},
		{
			name: "success - email send fails, continues without error",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).
					Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{
						{Email: "alice@example.com", Token: "tok-a"},
					}, nil).Once()
				f.emailService.On("SendNotification", mock.Anything, "alice@example.com", "golang/go", "v1.22.0", "tok-a").
					Return(errors.New("smtp error")).
					Once()
			},
		},
		{
			name: "success - CheckForUpdates fails, skips repo",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return((*model.Repository)(nil), errors.New("github error")).
					Once()
			},
		},
		{
			name: "success - GetByRepoID fails, skips repo",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).
					Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return(nil, errors.New("db error")).Once()
			},
		},
		{
			name: "error - GetAll fails",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name: "success - multiple repos, partial failures skipped",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
						{ID: 2, FullName: "torvalds/linux", LastSeenTag: "v6.8"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).
					Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, model.Repository{ID: 2, FullName: "torvalds/linux", LastSeenTag: "v6.8"}).
					Return((*model.Repository)(nil), errors.New("api error")).
					Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{
						{Email: "alice@example.com", Token: "tok-a"},
					}, nil).Once()
				f.emailService.On("SendNotification", mock.Anything, "alice@example.com", "golang/go", "v1.22.0", "tok-a").
					Return(nil).
					Once()
			},
		},
		{
			name: "success - send respects context timeout",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, mock.Anything).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{
						{Email: "alice@example.com", Token: "tok-a"},
					}, nil).Once()

				f.emailService.On("SendNotification",
					mock.MatchedBy(func(ctx context.Context) bool {
						_, hasDeadline := ctx.Deadline()
						return hasDeadline
					}),
					"alice@example.com", "golang/go", "v1.22.0", "tok-a",
				).Return(nil).Once()
			},
		},
		{
			name: "success - email send fails, warn logged, continues",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything,
					model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{
						{Email: "alice@example.com", Token: "tok-a"},
					}, nil).Once()
				f.emailService.On("SendNotification",
					mock.MatchedBy(func(ctx context.Context) bool {
						_, hasDeadline := ctx.Deadline()
						return hasDeadline
					}),
					"alice@example.com", "golang/go", "v1.22.0", "tok-a",
				).Return(errors.New("smtp error")).Once()
			},
			expectErr: false,
		},
		{
			name: "success - send succeeds, no warn logged",
			setup: func(f notifMockFields) {
				f.repoRepo.On("GetAll", mock.Anything).
					Return([]model.Repository{
						{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"},
					}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything,
					model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.21.0"}).
					Return(&model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.22.0"}, nil).Once()
				f.subsRepo.On("GetByRepoID", mock.Anything, int64(1)).
					Return([]model.Subscriber{
						{Email: "alice@example.com", Token: "tok-a"},
					}, nil).Once()

				f.emailService.On("SendNotification",
					mock.MatchedBy(func(ctx context.Context) bool {
						_, hasDeadline := ctx.Deadline()
						return hasDeadline
					}),
					"alice@example.com", "golang/go", "v1.22.0", "tok-a",
				).Return(nil).Once()
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newNotifMockFields(t)
			tt.setup(f)

			err := newNotifUC(f).ProcessNotifications(context.Background())

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNotificationUseCase_sendNotificationEmail(t *testing.T) {
	tests := []struct {
		name      string
		mockErr   error
		expectErr bool
	}{
		{
			name:      "success - email sent",
			mockErr:   nil,
			expectErr: false,
		},
		{
			name:      "error - smtp fails, error returned",
			mockErr:   errors.New("smtp error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newNotifMockFields(t)

			f.emailService.On("SendNotification",
				mock.MatchedBy(func(ctx context.Context) bool {
					deadline, hasDeadline := ctx.Deadline()
					if !hasDeadline {
						return false
					}
					remaining := time.Until(deadline)
					return remaining > (ctxTimeout-1)*time.Second && remaining <= ctxTimeout*time.Second
				}),
				"alice@example.com", "golang/go", "v1.22.0", "tok-a",
			).Return(tt.mockErr).Once()

			uc := newNotifUC(f)
			sub := model.Subscriber{Email: "alice@example.com", Token: "tok-a"}

			err := uc.sendNotificationEmail(context.Background(), sub, "golang/go", "v1.22.0")

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
