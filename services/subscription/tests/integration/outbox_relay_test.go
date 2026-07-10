package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stretchr/testify/mock"

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

	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(&pb.GetOrCreateRepositoryResponse{Id: 1, FullName: testRepo}, nil)

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	delivery := s.consumeOne(ch, contracts.QueueConfirmations, 10*time.Second)

	var cmd contracts.SendConfirmationCommand
	s.Require().NoError(json.Unmarshal(delivery.Body, &cmd))

	s.Positive(cmd.SagaID)
	s.Positive(cmd.SubscriptionID)
	s.Equal(testEmail, cmd.Email)
	s.Equal(testRepo, cmd.RepoName)
	s.NotEmpty(cmd.Token)
}
