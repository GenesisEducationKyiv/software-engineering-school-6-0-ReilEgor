package integration

import (
	"encoding/json"
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/dto"
)

func (s *APITestSuite) TestListSubscriptions_Success() {
	s.seedSubscription(testEmail, testRepo, testTag, true)

	w := s.doRequest(http.MethodGet, "/api/v1/subscriptions?email=test@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var resp dto.ListSubscriptionsResponse
	s.Require().NoError(json.NewDecoder(w.Body).Decode(&resp))
	s.Equal(1, resp.Total)
	s.Require().Len(resp.Subscriptions, 1)
	s.Equal(testRepo, resp.Subscriptions[0].RepositoryName)
	s.Equal(testEmail, resp.Subscriptions[0].Email)
	s.True(resp.Subscriptions[0].Confirmed)
}

func (s *APITestSuite) TestListSubscriptions_MultipleSubscriptions() {
	s.seedSubscription(testEmail, "golang/go", "v1.22.0", true)
	s.seedSubscription(testEmail, "kubernetes/kubernetes", "v1.30.0", false)

	w := s.doRequest(http.MethodGet, "/api/v1/subscriptions?email=test@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var resp dto.ListSubscriptionsResponse
	s.Require().NoError(json.NewDecoder(w.Body).Decode(&resp))
	s.Equal(2, resp.Total)
}

func (s *APITestSuite) TestListSubscriptions_EmptyForUnknownEmail() {
	w := s.doRequest(http.MethodGet, "/api/v1/subscriptions?email=nobody@example.com", nil)

	s.Equal(http.StatusOK, w.Code)

	var resp dto.ListSubscriptionsResponse
	s.Require().NoError(json.NewDecoder(w.Body).Decode(&resp))
	s.Equal(0, resp.Total)
	s.Empty(resp.Subscriptions)
}

func (s *APITestSuite) TestListSubscriptions_MissingEmailParam() {
	w := s.doRequest(http.MethodGet, "/api/v1/subscriptions", nil)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *APITestSuite) TestListSubscriptions_InvalidEmailParam() {
	w := s.doRequest(http.MethodGet, "/api/v1/subscriptions?email=not-an-email", nil)

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *APITestSuite) TestListSubscriptions_NoAPIKey() {
	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/subscriptions?email=test@example.com", nil)

	s.Equal(http.StatusUnauthorized, w.Code)
}
