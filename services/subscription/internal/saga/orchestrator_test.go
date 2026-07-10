package saga

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"

	sharedModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/mocks"
)

type orchestratorMockFields struct {
	sagaRepo   *mocks.SagaRepository
	subsRepo   *mocks.SubscriptionWriter
	outboxRepo *mocks.OutboxRepository
	transactor *mocks.Transactor
}

func newOrchestratorMockFields(t *testing.T) orchestratorMockFields {
	t.Helper()
	return orchestratorMockFields{
		sagaRepo:   mocks.NewSagaRepository(t),
		subsRepo:   mocks.NewSubscriptionWriter(t),
		outboxRepo: mocks.NewOutboxRepository(t),
		transactor: mocks.NewTransactor(t),
	}
}

func newTestOrchestrator(f orchestratorMockFields) *Orchestrator {
	return NewOrchestrator(f.sagaRepo, f.subsRepo, f.outboxRepo, f.transactor)
}

func setupTransactorOK(f orchestratorMockFields) {
	f.transactor.On("WithinTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn, ok := args.Get(1).(func(context.Context) error)
			if !ok {
				panic("unexpected argument type in WithinTransaction mock")
			}
			_ = fn(context.Background())
		}).
		Return(nil).Once()
}

func setupTransactorFail(f orchestratorMockFields, txErr error) {
	f.transactor.On("WithinTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn, ok := args.Get(1).(func(context.Context) error)
			if !ok {
				panic("unexpected argument type in WithinTransaction mock")
			}
			_ = fn(context.Background())
		}).
		Return(txErr).Once()
}

func TestOrchestrator_Start(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f orchestratorMockFields)
		expectErr bool
	}{
		{
			name: "success - saga started and confirmation command queued",
			setup: func(f orchestratorMockFields) {
				f.sagaRepo.On("Create", mock.Anything, int64(10)).
					Return(&sharedModel.SubscriptionSaga{ID: 1}, nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, contracts.QueueConfirmations,
					mock.MatchedBy(func(payload []byte) bool {
						var cmd contracts.SendConfirmationCommand
						if err := json.Unmarshal(payload, &cmd); err != nil {
							return false
						}
						return cmd.SagaID == 1 &&
							cmd.SubscriptionID == 10 &&
							cmd.Email == "user@example.com" &&
							cmd.RepoName == "golang/go" &&
							cmd.Token == "token-123"
					})).Return(nil).Once()
			},
		},
		{
			name: "error - create saga fails",
			setup: func(f orchestratorMockFields) {
				f.sagaRepo.On("Create", mock.Anything, int64(10)).
					Return((*sharedModel.SubscriptionSaga)(nil), errors.New("db error")).Once()
			},
			expectErr: true,
		},
		{
			name: "error - insert outbox fails",
			setup: func(f orchestratorMockFields) {
				f.sagaRepo.On("Create", mock.Anything, int64(10)).
					Return(&sharedModel.SubscriptionSaga{ID: 1}, nil).Once()
				f.outboxRepo.On("Insert", mock.Anything, contracts.QueueConfirmations, mock.AnythingOfType("[]uint8")).
					Return(errors.New("insert error")).Once()
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOrchestratorMockFields(t)
			tt.setup(f)

			err := newTestOrchestrator(f).Start(context.Background(), 10, "user@example.com", "golang/go", "token-123")

			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOrchestrator_HandleConfirmationReply(t *testing.T) {
	baseReply := contracts.ConfirmationResultEvent{
		SagaID:         1,
		SubscriptionID: 10,
		Email:          "user@example.com",
		RepoName:       "golang/go",
	}

	t.Run("success - activates subscription when confirmation succeeded", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		f.sagaRepo.On("GetByID", mock.Anything, int64(1)).
			Return(&sharedModel.SubscriptionSaga{ID: 1, Status: sharedModel.SagaStatusStarted}, nil).Once()
		f.sagaRepo.On("UpdateStatusAndStep", mock.Anything, int64(1),
			sharedModel.SagaStatusCompleted, sharedModel.SagaStepActivateSubscription).
			Return(nil).Once()

		reply := baseReply
		reply.Success = true

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.NoError(t, err)
	})

	t.Run("success - skips update when saga already completed", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		f.sagaRepo.On("GetByID", mock.Anything, int64(1)).
			Return(&sharedModel.SubscriptionSaga{ID: 1, Status: sharedModel.SagaStatusCompleted}, nil).Once()

		reply := baseReply
		reply.Success = true

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.NoError(t, err)
	})

	t.Run("error - get saga fails", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		f.sagaRepo.On("GetByID", mock.Anything, int64(1)).
			Return((*sharedModel.SubscriptionSaga)(nil), errors.New("db error")).Once()

		reply := baseReply
		reply.Success = true

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.Error(t, err)
	})

	t.Run("error - update status and step fails", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		f.sagaRepo.On("GetByID", mock.Anything, int64(1)).
			Return(&sharedModel.SubscriptionSaga{ID: 1, Status: sharedModel.SagaStatusStarted}, nil).Once()
		f.sagaRepo.On("UpdateStatusAndStep", mock.Anything, int64(1),
			sharedModel.SagaStatusCompleted, sharedModel.SagaStepActivateSubscription).
			Return(errors.New("update error")).Once()

		reply := baseReply
		reply.Success = true

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.Error(t, err)
	})

	t.Run("success - compensates subscription when confirmation failed", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		setupTransactorOK(f)
		f.subsRepo.On("DeleteByID", mock.Anything, int64(10)).
			Return(nil).Once()
		f.sagaRepo.On("UpdateStatus", mock.Anything, int64(1), sharedModel.SagaStatusCompensated).
			Return(nil).Once()

		reply := baseReply
		reply.Success = false

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.NoError(t, err)
	})

	t.Run("error - compensate delete subscription fails", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		deleteErr := errors.New("delete error")
		setupTransactorFail(f, deleteErr)
		f.subsRepo.On("DeleteByID", mock.Anything, int64(10)).
			Return(deleteErr).Once()

		reply := baseReply
		reply.Success = false

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.Error(t, err)
	})

	t.Run("error - compensate update saga status fails", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		updateErr := errors.New("update status error")
		setupTransactorFail(f, updateErr)
		f.subsRepo.On("DeleteByID", mock.Anything, int64(10)).
			Return(nil).Once()
		f.sagaRepo.On("UpdateStatus", mock.Anything, int64(1), sharedModel.SagaStatusCompensated).
			Return(updateErr).Once()

		reply := baseReply
		reply.Success = false

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.Error(t, err)
	})

	t.Run("error - transactor fails before running callback", func(t *testing.T) {
		f := newOrchestratorMockFields(t)
		txErr := errors.New("tx error")
		f.transactor.On("WithinTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(txErr).Once()

		reply := baseReply
		reply.Success = false

		err := newTestOrchestrator(f).HandleConfirmationReply(context.Background(), reply)

		require.Error(t, err)
	})
}
