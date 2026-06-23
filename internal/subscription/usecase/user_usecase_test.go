package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/domain/model"
	subMocks "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/mocks"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/subscription/saga"
	trackingModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/tracking/domain/model"
	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/domain/model"
	mocks2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/mocks"
)

type userMockFields struct {
	subsRepo   *subMocks.SubscriptionRepository
	userRepo   *mocks2.UserRepository
	repoUC     *mocks2.RepositoryUseCase
	sagaRepo   *subMocks.SagaRepository
	outboxRepo *subMocks.OutboxRepository
	transactor *subMocks.Transactor
}

func newUserMockFields(t *testing.T) userMockFields {
	t.Helper()
	return userMockFields{
		subsRepo:   subMocks.NewSubscriptionRepository(t),
		userRepo:   mocks2.NewUserRepository(t),
		repoUC:     mocks2.NewRepositoryUseCase(t),
		sagaRepo:   subMocks.NewSagaRepository(t),
		outboxRepo: subMocks.NewOutboxRepository(t),
		transactor: subMocks.NewTransactor(t),
	}
}

func newUserUC(f userMockFields) *UserUseCase {
	orchestrator := saga.NewOrchestrator(f.sagaRepo, f.subsRepo, f.outboxRepo, f.transactor)
	newUseUsecase, _ := NewUserUseCase(
		context.Background(),
		f.subsRepo,
		f.userRepo,
		f.repoUC,
		orchestrator,
		f.transactor,
		f.outboxRepo,
	)
	return newUseUsecase
}

// executes the callback directly, simulating a successful transaction.
func setupTransactorOK(f userMockFields) {
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

// simulates a transaction that rolls back and returns an error.
func setupTransactorFail(f userMockFields, txErr error) {
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
					Return(&trackingModel.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(subModel.User{ID: 10, Email: "user@example.com"}, nil).Once()
				setupTransactorOK(f)
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()
				f.sagaRepo.On("Create", mock.Anything, mock.AnythingOfType("int64")).
					Return(&sharedModel.SubscriptionSaga{ID: 1}, nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).
					Return(nil).Once()
			},
		},
		{
			name:     "success - new user created and subscribed",
			email:    "new@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&trackingModel.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "new@example.com").
					Return(subModel.User{}, subModel.ErrUserNotFound).Once()
				f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
					Run(func(args mock.Arguments) {
						u, ok := args.Get(1).(*subModel.User)
						if !ok {
							return
						}
						u.ID = 99
					}).Return(nil).Once()
				setupTransactorOK(f)
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()
				f.sagaRepo.On("Create", mock.Anything, mock.AnythingOfType("int64")).
					Return(&sharedModel.SubscriptionSaga{ID: 1}, nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).
					Return(nil).Once()
			},
		},
		{
			name:     "error - GetOrCreate fails",
			email:    "user@example.com",
			repoName: "unknown/repo",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "unknown/repo").
					Return((*trackingModel.Repository)(nil), errors.New("repo not found")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - GetByEmail unexpected DB error",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&trackingModel.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(subModel.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Create user fails",
			email:    "new@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&trackingModel.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "new@example.com").
					Return(subModel.User{}, subModel.ErrUserNotFound).Once()
				f.userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
					Return(errors.New("db write error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Save subscription fails, transaction rolled back",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&trackingModel.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(subModel.User{ID: 10, Email: "user@example.com"}, nil).Once()
				saveErr := errors.New("save error")
				setupTransactorFail(f, saveErr)
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(saveErr).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Create saga fails, subscription rolled back",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.repoUC.On("GetOrCreate", mock.Anything, "golang/go").
					Return(&trackingModel.Repository{ID: 1, FullName: "golang/go"}, nil).Once()
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(subModel.User{ID: 10, Email: "user@example.com"}, nil).Once()
				sagaErr := errors.New("saga create error")
				setupTransactorFail(f, sagaErr)
				f.subsRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Subscription")).
					Return(nil).Once()
				f.sagaRepo.On("Create", mock.Anything, mock.AnythingOfType("int64")).
					Return((*sharedModel.SubscriptionSaga)(nil), sagaErr).Once()
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
					Return(subModel.User{ID: 10, Email: "user@example.com"}, nil).Once()
				setupTransactorOK(f)
				f.subsRepo.On("Delete", mock.Anything, int64(10), "golang/go").
					Return(nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).
					Return(nil).Once()
			},
		},
		{
			name:     "success - user not found, nothing to unsubscribe",
			email:    "ghost@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "ghost@example.com").
					Return(subModel.User{}, subModel.ErrUserNotFound).Once()
			},
		},
		{
			name:     "error - GetByEmail unexpected DB error",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(subModel.User{}, errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name:     "error - Delete subscription fails",
			email:    "user@example.com",
			repoName: "golang/go",
			setup: func(f userMockFields) {
				deleteErr := errors.New("delete error")
				f.userRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return(subModel.User{ID: 10, Email: "user@example.com"}, nil).Once()
				setupTransactorFail(f, deleteErr)
				f.subsRepo.On("Delete", mock.Anything, int64(10), "golang/go").
					Return(deleteErr).Once()
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
		wantSubs  []subModel.Subscription
		expectErr bool
	}{
		{
			name:  "success - returns subscriptions",
			email: "user@example.com",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "user@example.com").
					Return([]subModel.Subscription{
						{UserID: 1, RepositoryName: "golang/go"},
						{UserID: 1, RepositoryName: "torvalds/linux"},
					}, nil).Once()
			},
			wantSubs: []subModel.Subscription{
				{UserID: 1, RepositoryName: "golang/go"},
				{UserID: 1, RepositoryName: "torvalds/linux"},
			},
		},
		{
			name:  "success - returns empty list",
			email: "empty@example.com",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByEmail", mock.Anything, "empty@example.com").
					Return([]subModel.Subscription{}, nil).Once()
			},
			wantSubs: []subModel.Subscription{},
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
				sub := &subModel.Subscription{
					UserID:         1,
					RepositoryName: "golang/go",
					Token:          "valid-token",
					Confirmed:      false,
				}
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").
					Return(sub, nil).Once()
				f.subsRepo.On("Save", mock.Anything, mock.MatchedBy(func(s *subModel.Subscription) bool {
					return s.Confirmed == true
				})).Return(nil).Once()
			},
		},
		{
			name:    "error - empty token",
			token:   "",
			setup:   func(_ userMockFields) {},
			wantErr: subModel.ErrInvalidToken,
		},
		{
			name:  "error - token not found",
			token: "bad-token",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "bad-token").
					Return((*subModel.Subscription)(nil), subModel.ErrInvalidToken).Once()
			},
			wantErr: subModel.ErrInvalidToken,
		},
		{
			name:  "error - Save fails after confirm",
			token: "valid-token",
			setup: func(f userMockFields) {
				sub := &subModel.Subscription{
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
					Return(&subModel.Subscription{UserID: 5, Email: "user@example.com", RepositoryName: "golang/go", Token: "valid-token"}, nil).
					Once()
				setupTransactorOK(f)
				f.subsRepo.On("Delete", mock.Anything, int64(5), "golang/go").
					Return(nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).
					Return(nil).Once()
			},
		},
		{
			name:    "error - empty token",
			token:   "",
			setup:   func(_ userMockFields) {},
			wantErr: subModel.ErrInvalidToken,
		},
		{
			name:  "error - token not found",
			token: "bad-token",
			setup: func(f userMockFields) {
				f.subsRepo.On("GetByToken", mock.Anything, "bad-token").
					Return((*subModel.Subscription)(nil), subModel.ErrInvalidToken).Once()
			},
			wantErr: subModel.ErrInvalidToken,
		},
		{
			name:  "error - Delete fails",
			token: "valid-token",
			setup: func(f userMockFields) {
				deleteErr := errors.New("delete error")
				f.subsRepo.On("GetByToken", mock.Anything, "valid-token").
					Return(&subModel.Subscription{UserID: 5, Email: "user@example.com", RepositoryName: "golang/go", Token: "valid-token"}, nil).
					Once()
				setupTransactorFail(f, deleteErr)
				f.subsRepo.On("Delete", mock.Anything, int64(5), "golang/go").
					Return(deleteErr).Once()
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
