package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/tracking/internal/mocks"
)

type releaseProcessorMockFields struct {
	repoReader    *mocks.RepositoryReader
	repoUC        *mocks.RepositoryUseCase
	subReader     *mocks.SubscriberReader
	publisher     *mocks.NotificationPublisher
	tagUpdatedPub *mocks.TagUpdatedPublisher
	outboxRepo    *mocks.OutboxRepository
	transactor    *mocks.Transactor
}

func newReleaseProcessorMockFields(t *testing.T) releaseProcessorMockFields {
	t.Helper()
	return releaseProcessorMockFields{
		repoReader:    mocks.NewRepositoryReader(t),
		repoUC:        mocks.NewRepositoryUseCase(t),
		subReader:     mocks.NewSubscriberReader(t),
		publisher:     mocks.NewNotificationPublisher(t),
		tagUpdatedPub: mocks.NewTagUpdatedPublisher(t),
		outboxRepo:    mocks.NewOutboxRepository(t),
		transactor:    mocks.NewTransactor(t),
	}
}

func newTestReleaseProcessor(f releaseProcessorMockFields) *ReleaseProcessor {
	return NewReleaseProcessor(
		f.repoReader,
		f.repoUC,
		f.subReader,
		f.publisher,
		f.tagUpdatedPub,
		f.outboxRepo,
		f.transactor,
	)
}

func setupTransactorOK(f releaseProcessorMockFields) {
	f.transactor.On("WithinTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn, ok := args.Get(1).(func(context.Context) error)
			if !ok {
				panic("unexpected argument type in WithinTransaction mock")
			}
			if err := fn(context.Background()); err != nil {
				return
			}
		}).
		Return(nil).Once()
}

func setupTransactorFail(f releaseProcessorMockFields, txErr error) {
	f.transactor.On("WithinTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn, ok := args.Get(1).(func(context.Context) error)
			if !ok {
				panic("unexpected argument type in WithinTransaction mock")
			}
			if err := fn(context.Background()); err != nil {
				return
			}
		}).
		Return(txErr).Once()
}

func notificationPayloadMatcher(sub model.Subscriber, repo *model.Repository) interface{} {
	return mock.MatchedBy(func(payload []byte) bool {
		var cmd contracts.SendNotificationCommand
		if err := json.Unmarshal(payload, &cmd); err != nil {
			return false
		}
		return cmd.Email == sub.Email &&
			cmd.Token == sub.Token &&
			cmd.RepoName == repo.FullName &&
			cmd.Tag == repo.LastSeenTag
	})
}

func TestReleaseProcessor_ProcessReleases(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f releaseProcessorMockFields)
		expectErr bool
	}{
		{
			name: "error - get all repos fails",
			setup: func(f releaseProcessorMockFields) {
				f.repoReader.On("GetAll", mock.Anything).
					Return(nil, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name: "success - no repos to process",
			setup: func(f releaseProcessorMockFields) {
				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{}, nil).Once()
			},
		},
		{
			name: "success - repo has no update, skipped",
			setup: func(f releaseProcessorMockFields) {
				repo := model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.0.0"}
				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repo}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repo).
					Return((*model.Repository)(nil), nil).Once()
			},
		},
		{
			name: "success - check for updates fails, repo error is swallowed",
			setup: func(f releaseProcessorMockFields) {
				repo := model.Repository{ID: 1, FullName: "golang/go"}
				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repo}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repo).
					Return((*model.Repository)(nil), errors.New("github error")).Once()
			},
		},
		{
			name: "success - get subscribers fails, repo error is swallowed",
			setup: func(f releaseProcessorMockFields) {
				repo := model.Repository{ID: 1, FullName: "golang/go"}
				updated := &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v2.0.0"}
				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repo}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repo).
					Return(updated, nil).Once()
				f.subReader.On("GetByRepoID", mock.Anything, int64(1)).
					Return(nil, errors.New("db error")).Once()
			},
		},
		{
			name: "success - release committed and tag update published",
			setup: func(f releaseProcessorMockFields) {
				repo := model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v1.0.0"}
				updated := &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v2.0.0"}
				subs := []model.Subscriber{
					{Email: "a@example.com", Token: "tok-a"},
					{Email: "b@example.com", Token: "tok-b"},
				}

				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repo}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repo).
					Return(updated, nil).Once()
				f.subReader.On("GetByRepoID", mock.Anything, int64(1)).
					Return(subs, nil).Once()
				setupTransactorOK(f)
				f.repoUC.On("UpdateRepo", mock.Anything, updated).
					Return(nil).Once()
				for _, sub := range subs {
					f.outboxRepo.On("Insert", mock.Anything, contracts.QueueNotifications, notificationPayloadMatcher(sub, updated)).
						Return(nil).
						Once()
				}
				f.tagUpdatedPub.On("Publish", mock.Anything, contracts.TagUpdatedEvent{
					FullName: updated.FullName,
					Tag:      updated.LastSeenTag,
				}).Return(nil).Once()
			},
		},
		{
			name: "success - commit fails, repo error swallowed and tag update not published",
			setup: func(f releaseProcessorMockFields) {
				repo := model.Repository{ID: 1, FullName: "golang/go"}
				updated := &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v2.0.0"}
				subs := []model.Subscriber{{Email: "a@example.com", Token: "tok-a"}}
				updateErr := errors.New("update repo failed")

				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repo}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repo).
					Return(updated, nil).Once()
				f.subReader.On("GetByRepoID", mock.Anything, int64(1)).
					Return(subs, nil).Once()
				setupTransactorFail(f, updateErr)
				f.repoUC.On("UpdateRepo", mock.Anything, updated).
					Return(updateErr).Once()
			},
		},
		{
			name: "success - tag update publish fails, error only logged",
			setup: func(f releaseProcessorMockFields) {
				repo := model.Repository{ID: 1, FullName: "golang/go"}
				updated := &model.Repository{ID: 1, FullName: "golang/go", LastSeenTag: "v2.0.0"}
				subs := []model.Subscriber{{Email: "a@example.com", Token: "tok-a"}}

				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repo}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repo).
					Return(updated, nil).Once()
				f.subReader.On("GetByRepoID", mock.Anything, int64(1)).
					Return(subs, nil).Once()
				setupTransactorOK(f)
				f.repoUC.On("UpdateRepo", mock.Anything, updated).
					Return(nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, contracts.QueueNotifications, notificationPayloadMatcher(subs[0], updated)).
					Return(nil).
					Once()
				f.tagUpdatedPub.On("Publish", mock.Anything, contracts.TagUpdatedEvent{
					FullName: updated.FullName,
					Tag:      updated.LastSeenTag,
				}).Return(errors.New("publish error")).Once()
			},
		},
		{
			name: "success - loop continues after one repo fails",
			setup: func(f releaseProcessorMockFields) {
				repoA := model.Repository{ID: 1, FullName: "foo/a"}
				repoB := model.Repository{ID: 2, FullName: "foo/b", LastSeenTag: "v1.0.0"}
				updatedB := &model.Repository{ID: 2, FullName: "foo/b", LastSeenTag: "v2.0.0"}

				f.repoReader.On("GetAll", mock.Anything).
					Return([]model.Repository{repoA, repoB}, nil).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repoA).
					Return((*model.Repository)(nil), errors.New("github rate limited")).Once()
				f.repoUC.On("CheckForUpdates", mock.Anything, repoB).
					Return(updatedB, nil).Once()
				f.subReader.On("GetByRepoID", mock.Anything, int64(2)).
					Return([]model.Subscriber{}, nil).Once()
				setupTransactorOK(f)
				f.repoUC.On("UpdateRepo", mock.Anything, updatedB).
					Return(nil).Once()
				f.tagUpdatedPub.On("Publish", mock.Anything, contracts.TagUpdatedEvent{
					FullName: updatedB.FullName,
					Tag:      updatedB.LastSeenTag,
				}).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReleaseProcessorMockFields(t)
			tt.setup(f)

			err := newTestReleaseProcessor(f).ProcessReleases(context.Background())

			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
