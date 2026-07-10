package integration

import "net/http"

func (s *APITestSuite) TestConfirm_Success() {
	token := s.seedSubscription(testEmail, testRepo, testTag, false)

	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/confirm/"+token, nil)

	s.Equal(http.StatusOK, w.Code)

	var confirmed bool
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT is_confirmed FROM subscriptions WHERE token = $1`, token,
	).Scan(&confirmed))
	s.True(confirmed)
}

func (s *APITestSuite) TestConfirm_AlreadyConfirmed() {
	token := s.seedSubscription(testEmail, testRepo, testTag, true)

	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/confirm/"+token, nil)

	s.Equal(http.StatusOK, w.Code)
}

func (s *APITestSuite) TestConfirm_InvalidToken() {
	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/confirm/non-existent-token", nil)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *APITestSuite) TestUnsubscribeByToken_Success() {
	token := s.seedSubscription(testEmail, testRepo, testTag, true)

	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/unsubscribe/"+token, nil)

	s.Equal(http.StatusOK, w.Code)

	var count int
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE token = $1`, token,
	).Scan(&count))
	s.Equal(0, count)
}

func (s *APITestSuite) TestUnsubscribeByToken_InvalidToken() {
	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/unsubscribe/non-existent-token", nil)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *APITestSuite) TestUnsubscribeByToken_PendingSubscription() {
	token := s.seedSubscription(testEmail, testRepo, testTag, false)

	w := s.doRequestNoAuth(http.MethodGet, "/api/v1/unsubscribe/"+token, nil)

	s.Equal(http.StatusOK, w.Code)

	var count int
	s.Require().NoError(s.dbPool.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE token = $1`, token,
	).Scan(&count))
	s.Equal(0, count)
}
