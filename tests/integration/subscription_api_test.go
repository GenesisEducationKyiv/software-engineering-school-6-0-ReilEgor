package integration

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stretchr/testify/mock"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/dto"
)

func (s *APITestSuite) TestSubscribe_Success() {
	const (
		email     = "test@example.com"
		repoName  = "golang/go"
		latestTag = "v1.22.0"
	)

	s.mockGitHub.On("RepoExists", mock.Anything, repoName).Return(true, nil)

	s.mockGitHub.On("GetLatestRelease", mock.Anything, repoName).
		Return(&model.ReleaseInfo{TagName: latestTag}, nil)

	s.mockSMTP.On("SendConfirmation", mock.Anything, email, repoName, mock.AnythingOfType("string")).
		Return(nil).Maybe()

	body := `{"email":"test@example.com","repository":"golang/go"}`

	w := s.doRequest(http.MethodPost, "/api/v1/subscribe", strings.NewReader(body))

	s.Equal(http.StatusAccepted, w.Code)

	var resp dto.CreateSubscriptionResponse
	s.Require().NoError(json.NewDecoder(w.Body).Decode(&resp))
	s.NotEmpty(resp.Message)

	var count int
	err := s.dbPool.QueryRow(s.ctx,
		`SELECT COUNT(*) FROM subscriptions s
		 JOIN users u ON u.id = s.user_id
		 JOIN repositories r ON r.id = s.repository_id
		 WHERE u.email = $1 AND r.full_name = $2 AND s.is_confirmed = false`,
		email, repoName,
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count, "there must be exactly one pending subscription in the database")

	var savedTag string
	err = s.dbPool.QueryRow(s.ctx,
		`SELECT last_seen_tag FROM repositories WHERE full_name = $1`, repoName,
	).Scan(&savedTag)
	s.Require().NoError(err)
	s.Equal(latestTag, savedTag)
}
