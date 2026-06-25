package integration

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/transport/http/dto"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/infrastructure/grpc/proto/v1"
)

func (s *APITestSuite) TestSubscribe_Success() {
	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(&pb.GetOrCreateRepositoryResponse{Id: 1, FullName: testRepo}, nil)
	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))

	s.Equal(http.StatusAccepted, w.Code)

	var resp dto.CreateSubscriptionResponse
	s.Require().NoError(json.NewDecoder(w.Body).Decode(&resp))
	s.NotEmpty(resp.Message)

	var count int
	s.Require().NoError(s.dbPool.QueryRow(s.ctx, `
		SELECT COUNT(*) FROM subscriptions s
		JOIN users u ON u.id = s.user_id
		JOIN repositories r ON r.id = s.repository_id
		WHERE u.email = $1 AND r.full_name = $2 AND s.is_confirmed = false`,
		testEmail, testRepo,
	).Scan(&count))
	s.Equal(1, count)
}

func (s *APITestSuite) TestSubscribe_InvalidEmail() {
	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"not-an-email","repository":"golang/go"}`))

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *APITestSuite) TestSubscribe_InvalidRepoFormat() {
	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"invalid-repo-without-slash"}`))

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *APITestSuite) TestSubscribe_EmptyBody() {
	w := s.doRequest(http.MethodPost, "/api/v1/subscribe", strings.NewReader(`{}`))

	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *APITestSuite) TestSubscribe_RepoNotFoundOnGitHub() {
	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(nil, status.Error(codes.NotFound, "repository not found"))

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *APITestSuite) TestSubscribe_GitHubUnavailable() {
	s.mockTracking.On("GetOrCreateRepository", mock.Anything, &pb.GetOrCreateRepositoryRequest{FullName: testRepo}).
		Return(nil, status.Error(codes.Unavailable, "external service unavailable"))

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))

	s.Equal(http.StatusServiceUnavailable, w.Code)
}

func (s *APITestSuite) TestSubscribe_NoAPIKey() {
	w := s.doRequestNoAuth(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))

	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *APITestSuite) TestSubscribe_WrongAPIKey() {
	w := s.doRequestWithKey(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`),
		"wrong-key",
	)

	s.Equal(http.StatusUnauthorized, w.Code)
}
