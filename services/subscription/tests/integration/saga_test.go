package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/saga"
	"github.com/stretchr/testify/mock"

	subModel "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/model"
	subOutbox "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/infrastructure/outbox"
	subPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/repository/postgres"
	subRabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/broker/rabbitmq"
	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
	sharedPostgres "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/storage/postgres"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (s *APITestSuite) startSagaResultConsumer() {
	s.T().Helper()
	sagaRepo := subPostgres.NewSagaRepository(s.dbPool)
	subsRepo := subPostgres.NewSubscriptionRepository(s.dbPool)
	outboxRepo := subPostgres.NewOutboxRepository(s.dbPool)
	transactor := sharedPostgres.NewTransactor(s.dbPool)
	orchestrator := saga.NewOrchestrator(sagaRepo, subsRepo, outboxRepo, transactor)

	consumer := subRabbitmq.NewSagaResultConsumer(s.rabbitConn, orchestrator)

	ctx, cancel := context.WithCancel(s.ctx)
	s.T().Cleanup(cancel)

	go func() {
		if err := consumer.Start(ctx); err != nil {
			s.T().Logf("saga result consumer stopped: %v", err)
		}
	}()
}

func (s *APITestSuite) publishConfirmationResult(event contracts.ConfirmationResultEvent) {
	s.T().Helper()
	ch, err := s.rabbitConn.Channel()
	s.Require().NoError(err)
	defer func() {
		if err := ch.Close(); err != nil {
			s.T().Logf("close channel: %v", err)
		}
	}()

	body, err := json.Marshal(event)
	s.Require().NoError(err)

	s.Require().
		NoError(ch.PublishWithContext(s.ctx, "", contracts.QueueSagaConfirmationResults, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}))
}

func (s *APITestSuite) sagaStatus(subscriptionID int64) string {
	s.T().Helper()
	var status string
	err := s.dbPool.QueryRow(s.ctx,
		`SELECT status FROM subscription_sagas WHERE subscription_id = $1 ORDER BY id DESC LIMIT 1`,
		subscriptionID,
	).Scan(&status)
	s.Require().NoError(err)
	return status
}

func (s *APITestSuite) subscribeAndAwaitConfirmationCommand() contracts.SendConfirmationCommand {
	s.T().Helper()

	confirmCh := s.declareAndPurgeQueue(contracts.QueueConfirmations)
	defer func() {
		if err := confirmCh.Close(); err != nil {
			s.T().Logf("close channel: %v", err)
		}
	}()
	resultsCh := s.declareAndPurgeQueue(contracts.QueueSagaConfirmationResults)
	defer func() {
		if err := resultsCh.Close(); err != nil {
			s.T().Logf("close channel: %v", err)
		}
	}()

	s.startOutboxRelay(subOutbox.NewRabbitPublisher(s.rabbitConn), outboxRelayInterval)
	s.startSagaResultConsumer()

	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(&pb.GetOrCreateRepositoryResponse{Id: 1, FullName: testRepo}, nil)

	confirmMsgs := s.consume(confirmCh, contracts.QueueConfirmations)

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	delivery := s.awaitDelivery(confirmMsgs, 10*time.Second)

	var cmd contracts.SendConfirmationCommand
	s.Require().NoError(json.Unmarshal(delivery.Body, &cmd))
	return cmd
}

func (s *APITestSuite) TestSagaRoundTrip_ConfirmationSucceeded_CompletesSaga() {
	cmd := s.subscribeAndAwaitConfirmationCommand()

	s.publishConfirmationResult(contracts.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        true,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
	})

	s.Require().Eventually(func() bool {
		return s.sagaStatus(cmd.SubscriptionID) == string(subModel.SagaStatusCompleted)
	}, 5*time.Second, 50*time.Millisecond)

	var confirmed bool
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT is_confirmed FROM subscriptions WHERE id = $1`, cmd.SubscriptionID,
	).Scan(&confirmed))
	s.False(confirmed)
}

func (s *APITestSuite) TestSagaRoundTrip_ConfirmationFailed_CompensatesSubscription() {
	cmd := s.subscribeAndAwaitConfirmationCommand()

	s.publishConfirmationResult(contracts.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        false,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
	})

	s.Require().Eventually(func() bool {
		return s.sagaStatus(cmd.SubscriptionID) == string(subModel.SagaStatusCompensated)
	}, 5*time.Second, 50*time.Millisecond)

	var count int
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE id = $1`, cmd.SubscriptionID,
	).Scan(&count))
	s.Equal(0, count)
}

func (s *APITestSuite) TestSagaRoundTrip_DuplicateConfirmationResult_Idempotent() {
	cmd := s.subscribeAndAwaitConfirmationCommand()

	successEvent := contracts.ConfirmationResultEvent{
		SagaID:         cmd.SagaID,
		SubscriptionID: cmd.SubscriptionID,
		Success:        true,
		Email:          cmd.Email,
		RepoName:       cmd.RepoName,
		Token:          cmd.Token,
	}

	s.publishConfirmationResult(successEvent)
	s.Require().Eventually(func() bool {
		return s.sagaStatus(cmd.SubscriptionID) == string(subModel.SagaStatusCompleted)
	}, 5*time.Second, 50*time.Millisecond)

	s.publishConfirmationResult(successEvent)
	time.Sleep(300 * time.Millisecond)

	s.Equal(string(subModel.SagaStatusCompleted), s.sagaStatus(cmd.SubscriptionID))
}
