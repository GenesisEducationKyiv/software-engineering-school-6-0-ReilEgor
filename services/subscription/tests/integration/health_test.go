package integration

import (
	"encoding/json"
	"net/http"
)

func (s *APITestSuite) TestHealth() {
	w := s.doRequestNoAuth(http.MethodGet, "/health", nil)

	s.Equal(http.StatusOK, w.Code)

	var resp map[string]string
	s.Require().NoError(json.NewDecoder(w.Body).Decode(&resp))
	s.Equal("ok", resp["status"])
}
