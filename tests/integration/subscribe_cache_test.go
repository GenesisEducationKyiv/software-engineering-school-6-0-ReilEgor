package integration

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stretchr/testify/mock"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
)

func (s *APITestSuite) TestSubscribe_WritesRepoExistsToRedis() {
	s.mockGitHub.On("RepoExists", mock.Anything, "golang/go").Return(true, nil).Once()
	s.mockGitHub.On("GetLatestRelease", mock.Anything, "golang/go").
		Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
	s.mockSMTP.On("SendConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	val, err := s.redisClient.Get(s.ctx, "repo_exists:golang/go").Result()
	s.Require().NoError(err, "repo_exists key must be in Redis after subscribe")
	s.Equal("true", val)
}

func (s *APITestSuite) TestSubscribe_WritesLatestReleaseToRedis() {
	s.mockGitHub.On("RepoExists", mock.Anything, "golang/go").Return(true, nil).Once()
	s.mockGitHub.On("GetLatestRelease", mock.Anything, "golang/go").
		Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
	s.mockSMTP.On("SendConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	raw, err := s.redisClient.Get(s.ctx, "release:golang/go").Result()
	s.Require().NoError(err, "release key must be in Redis after subscribe")

	var cached model.ReleaseInfo
	s.Require().NoError(json.Unmarshal([]byte(raw), &cached))
	s.Equal("v1.22.0", cached.TagName)
}

func (s *APITestSuite) TestSubscribe_SecondCall_HitsCache() {
	s.mockGitHub.On("RepoExists", mock.Anything, "golang/go").Return(true, nil).Once()
	s.mockGitHub.On("GetLatestRelease", mock.Anything, "golang/go").
		Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()
	s.mockSMTP.On("SendConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"first@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)

	w = s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"second@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w.Code)
}

func (s *APITestSuite) TestSubscribe_CacheFlush_UsesDatabaseFallback() {
	s.mockGitHub.On("RepoExists", mock.Anything, "golang/go").
		Return(true, nil).Once()
	s.mockGitHub.On("GetLatestRelease", mock.Anything, "golang/go").
		Return(&model.ReleaseInfo{TagName: "v1.22.0"}, nil).Once()

	s.mockSMTP.On("SendConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()

	w1 := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"first@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w1.Code)

	s.Require().NoError(s.redisClient.FlushAll(s.ctx).Err())

	w2 := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"second@example.com","repository":"golang/go"}`))
	s.Require().Equal(http.StatusAccepted, w2.Code)

	var subsCount, repoCount int
	err := s.dbPool.QueryRow(s.ctx, "SELECT COUNT(*) FROM subscriptions").Scan(&subsCount)
	s.Require().NoError(err)
	s.Require().Equal(2, subsCount, "Expected 2 subscriptions in the database")

	err = s.dbPool.QueryRow(s.ctx, "SELECT COUNT(*) FROM repositories").Scan(&repoCount)
	s.Require().NoError(err)
	s.Require().Equal(1, repoCount, "Only 1 stored repository in the database was expected")
}

func (s *APITestSuite) TestSubscribe_RepoNotFound_NothingCached() {
	s.mockGitHub.On("RepoExists", mock.Anything, "ghost/nope").Return(false, nil).Once()

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe",
		strings.NewReader(`{"email":"test@example.com","repository":"ghost/nope"}`))
	s.Require().Equal(http.StatusNotFound, w.Code)

	val, err := s.redisClient.Get(s.ctx, "repo_exists:ghost/nope").Result()
	s.Require().NoError(err, "false result must still be cached to prevent repeated GitHub calls")
	s.Equal("false", val)

	keys, err := s.redisClient.Keys(s.ctx, "release:*").Result()
	s.Require().NoError(err)
	s.Empty(keys, "release key must not be cached when repo does not exist")
}
