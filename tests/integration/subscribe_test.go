package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stretchr/testify/mock"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/dto"
)

func (s *APITestSuite) TestSubscribe_Success() {
	s.mockGitHub.On("RepoExists", mock.Anything, testRepo).Return(true, nil)
	s.mockGitHub.On("GetLatestRelease", mock.Anything, testRepo).
		Return(&model.ReleaseInfo{TagName: testTag}, nil)
	s.mockSMTP.On("SendConfirmation", mock.Anything, testEmail, testRepo, mock.AnythingOfType("string")).
		Return(nil).Maybe()

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

	var savedTag string
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT last_seen_tag FROM repositories WHERE full_name = $1`, testRepo,
	).Scan(&savedTag))
	s.Equal(testTag, savedTag)
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
	s.mockGitHub.On("RepoExists", mock.Anything, testRepo).Return(false, nil)

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *APITestSuite) TestSubscribe_GitHubUnavailable() {
	s.mockGitHub.On("RepoExists", mock.Anything, testRepo).
		Return(false, fmt.Errorf("wrapped: %w", model.ErrRepositoryNotFound))

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))

	s.GreaterOrEqual(w.Code, http.StatusBadRequest)
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
