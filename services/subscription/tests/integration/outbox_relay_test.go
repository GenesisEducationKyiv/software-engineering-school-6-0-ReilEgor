package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stretchr/testify/mock"

	subOutbox "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/infrastructure/outbox"
	contracts "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/contracts"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

func (s *APITestSuite) TestSubscribe_PublishesConfirmationCommandToQueue() {
	ch := s.declareAndPurgeQueue(contracts.QueueConfirmations)
	defer func() {
		if err := ch.Close(); err != nil {
			s.T().Logf("close channel: %v", err)
		}
	}()
	s.startOutboxRelay(subOutbox.NewRabbitPublisher(s.rabbitConn), outboxRelayInterval)

	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(&pb.GetOrCreateRepositoryResponse{Id: 1, FullName: testRepo}, nil)

	msgs := s.consume(ch, contracts.QueueConfirmations)

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	delivery := s.awaitDelivery(msgs, 10*time.Second)

	var cmd contracts.SendConfirmationCommand
	s.Require().NoError(json.Unmarshal(delivery.Body, &cmd))

	s.Positive(cmd.SagaID)
	s.Positive(cmd.SubscriptionID)
	s.Equal(testEmail, cmd.Email)
	s.Equal(testRepo, cmd.RepoName)
	s.NotEmpty(cmd.Token)
}

func (s *APITestSuite) TestSubscribe_MultipleInARow_PreservesOrderNoLossNoDuplicates() {
	ch := s.declareAndPurgeQueue(contracts.QueueConfirmations)
	defer func() {
		if err := ch.Close(); err != nil {
			s.T().Logf("close channel: %v", err)
		}
	}()

	s.startOutboxRelay(subOutbox.NewRabbitPublisher(s.rabbitConn), outboxRelayInterval)

	repos := []string{"golang/go", "torvalds/linux", "kubernetes/kubernetes"}
	for i, repo := range repos {
		s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: repo}).
			Return(&pb.GetOrCreateRepositoryResponse{Id: int64(i + 1), FullName: repo}, nil)
	}

	msgs := s.consume(ch, contracts.QueueConfirmations)

	for _, repo := range repos {
		w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
			strings.NewReader(`{"email":"test@example.com","repository":"`+repo+`"}`))
		s.Require().Equal(http.StatusAccepted, w.Code)
	}

	for _, wantRepo := range repos {
		delivery := s.awaitDelivery(msgs, 10*time.Second)

		var cmd contracts.SendConfirmationCommand
		s.Require().NoError(json.Unmarshal(delivery.Body, &cmd))
		s.Equal(wantRepo, cmd.RepoName)
	}

	s.assertNoMoreDeliveries(msgs, time.Second)
}

type flakyPublisher struct {
	real    subOutbox.Publisher
	calls   int32
	failFor int32
}

func (p *flakyPublisher) Publish(ctx context.Context, queue string, body []byte) error {
	if atomic.AddInt32(&p.calls, 1) <= p.failFor {
		return errors.New("simulated transient publish failure")
	}
	if err := p.real.Publish(ctx, queue, body); err != nil {
		return fmt.Errorf("flaky publisher: %w", err)
	}
	return nil
}

func (s *APITestSuite) TestSubscribe_OutboxRetriesAfterTransientPublishFailure() {
	ch := s.declareAndPurgeQueue(contracts.QueueConfirmations)
	defer func() {
		if err := ch.Close(); err != nil {
			s.T().Logf("close channel: %v", err)
		}
	}()

	publisher := &flakyPublisher{real: subOutbox.NewRabbitPublisher(s.rabbitConn), failFor: 1}
	s.startOutboxRelay(publisher, 300*time.Millisecond)

	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(&pb.GetOrCreateRepositoryResponse{Id: 1, FullName: testRepo}, nil)

	msgs := s.consume(ch, contracts.QueueConfirmations)

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	s.Require().Eventually(func() bool {
		var attempts int
		var status string
		err := s.dbPool.QueryRow(s.ctx,
			`SELECT attempts, status FROM outbox_messages WHERE queue = $1`,
			contracts.QueueConfirmations,
		).Scan(&attempts, &status)
		return err == nil && attempts == 1 && status == "PENDING"
	}, 3*time.Second, 20*time.Millisecond, "expected the first publish attempt to fail and be recorded")

	delivery := s.awaitDelivery(msgs, 10*time.Second)

	var cmd contracts.SendConfirmationCommand
	s.Require().NoError(json.Unmarshal(delivery.Body, &cmd))
	s.Equal(testEmail, cmd.Email)
	s.Equal(testRepo, cmd.RepoName)

	s.GreaterOrEqual(atomic.LoadInt32(&publisher.calls), int32(2))

	var count int
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM outbox_messages WHERE queue = $1`, contracts.QueueConfirmations,
	).Scan(&count))
	s.Equal(0, count)
}
