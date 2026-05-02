package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
	"github.com/ReilEgor/RepoNotifier/internal/mocks"
	"github.com/ReilEgor/RepoNotifier/internal/transport/http/dto"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestHandler(uc *mocks.SubscriptionUseCase) *Handler {
	return &Handler{
		subscriptionUC: uc,
		logger:         discardLogger(),
	}
}

func newRouter(method, path string, handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Handle(method, path, handler)
	return r
}

func doRequest(r *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
	var b *bytes.Buffer
	if body != nil {
		raw, _ := json.Marshal(body)
		b = bytes.NewBuffer(raw)
	} else {
		b = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, url, b)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Subscribe(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func(uc *mocks.SubscriptionUseCase)
		expectedStatus int
		checkBody      func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "success — 202 accepted",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").Return(nil).Once()
			},
			expectedStatus: http.StatusAccepted,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["message"], "email to confirm")
			},
		},
		{
			name:           "invalid json body",
			body:           "not-json",
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing email",
			body:           map[string]string{"repository": "golang/go"},
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				_, hasError := body["error"]
				_, hasErrors := body["errors"]
				assert.True(t, hasError || hasErrors, "Response should contain error or errors key")
			},
		},
		{
			name:           "invalid email format",
			body:           map[string]string{"email": "not-an-email", "repository": "golang/go"},
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				if errs, ok := body["errors"].([]interface{}); ok {
					assert.NotEmpty(t, errs)
					assert.Contains(t, errs[0], "invalid email")
				} else if errMsg, ok := body["error"].(string); ok {
					assert.Contains(t, errMsg, "invalid")
				} else {
					t.Fatal("Response body does not contain expected error fields")
				}
			},
		},
		{
			name:           "invalid repo format — no slash",
			body:           map[string]string{"email": "test@example.com", "repository": "badrepo"},
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				errs := body["errors"].([]interface{})
				assert.Contains(t, errs[0], "owner/repo")
			},
		},
		{
			name:           "both email and repo invalid — two errors returned",
			body:           map[string]string{"email": "bad", "repository": "bad"},
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				val, ok := body["errors"]
				if !ok {
					errMsg, hasErr := body["error"]
					assert.True(t, hasErr, "Response should have 'errors' or 'error' key")
					fmt.Printf("Actual error received: %v\n", errMsg)
					return
				}

				errs, isSlice := val.([]interface{})
				assert.True(t, isSlice, "errors should be a slice")
				assert.GreaterOrEqual(t, len(errs), 1)
			},
		},
		{
			name: "repository not found",
			body: map[string]string{"email": "test@example.com", "repository": "owner/repo"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "owner/repo").
					Return(service.ErrRepositoryNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "already subscribed — 409 conflict",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(ErrAlreadySubscribed).Once()
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "github unavailable — 503",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(service.ErrGitHubUnavailable).Once()
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "rate limit exceeded — 503",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(service.ErrRateLimitExceeded).Once()
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "unexpected internal error — 500",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(fmt.Errorf("unexpected db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := mocks.NewSubscriptionUseCase(t)
			tt.mockSetup(mockUC)

			r := newRouter(http.MethodPost, "/subscribe", newTestHandler(mockUC).Subscribe)
			w := doRequest(r, http.MethodPost, "/subscribe", tt.body)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkBody != nil {
				var body map[string]interface{}
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkBody(t, body)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandler_Confirm(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		mockSetup      func(uc *mocks.SubscriptionUseCase)
		expectedStatus int
		checkBody      func(t *testing.T, body map[string]interface{})
	}{
		{
			name:  "success — 200",
			token: "valid_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Confirm", mock.Anything, "valid_token").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["message"], "confirmed")
			},
		},
		{
			name:  "invalid token — 404",
			token: "bad_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Confirm", mock.Anything, "bad_token").Return(model.ErrInvalidToken).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "expired")
			},
		},
		{
			name:  "internal error — 500",
			token: "some_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Confirm", mock.Anything, "some_token").
					Return(fmt.Errorf("db failure")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := mocks.NewSubscriptionUseCase(t)
			tt.mockSetup(mockUC)

			r := newRouter(http.MethodGet, "/confirm/:token", newTestHandler(mockUC).Confirm)
			w := doRequest(r, http.MethodGet, "/confirm/"+tt.token, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkBody != nil {
				var body map[string]interface{}
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkBody(t, body)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandler_UnsubscribeByToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		mockSetup      func(uc *mocks.SubscriptionUseCase)
		expectedStatus int
		checkBody      func(t *testing.T, body map[string]interface{})
	}{
		{
			name:  "success — 200",
			token: "token123",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("UnsubscribeByToken", mock.Anything, "token123").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["message"], "unsubscribed")
			},
		},
		{
			name:  "invalid token — 404",
			token: "expired_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("UnsubscribeByToken", mock.Anything, "expired_token").
					Return(model.ErrInvalidToken).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "expired")
			},
		},
		{
			name:  "internal error — 500",
			token: "some_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("UnsubscribeByToken", mock.Anything, "some_token").
					Return(fmt.Errorf("db crash")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := mocks.NewSubscriptionUseCase(t)
			tt.mockSetup(mockUC)

			r := newRouter(http.MethodGet, "/unsubscribe/:token", newTestHandler(mockUC).UnsubscribeByToken)
			w := doRequest(r, http.MethodGet, "/unsubscribe/"+tt.token, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkBody != nil {
				var body map[string]interface{}
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkBody(t, body)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandler_ListSubscriptions(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockSetup      func(uc *mocks.SubscriptionUseCase)
		expectedStatus int
		checkBody      func(t *testing.T, body map[string]interface{})
	}{
		{
			name:  "success — returns list",
			query: "?email=test@example.com",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "test@example.com").Return([]model.Subscription{
					{ID: 1, RepositoryName: "golang/go", Confirmed: true},
					{ID: 2, RepositoryName: "google/uuid", Confirmed: false},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(2), body["total"])
				subs := body["subscriptions"].([]interface{})
				assert.Len(t, subs, 2)
			},
		},
		{
			name:           "missing email — 400",
			query:          "",
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "email")
			},
		},
		{
			name:           "invalid email format — 400",
			query:          "?email=not-valid",
			mockSetup:      func(uc *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "invalid email")
			},
		},
		{
			name:  "empty subscription list — 200 with empty array",
			query: "?email=new@example.com",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "new@example.com").
					Return([]model.Subscription{}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(0), body["total"])
				subs := body["subscriptions"].([]interface{})
				assert.Empty(t, subs)
			},
		},
		{
			name:  "usecase error — 500",
			query: "?email=test@example.com",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "test@example.com").
					Return(nil, fmt.Errorf("db connection lost")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:  "email with leading/trailing spaces — trimmed correctly",
			query: "?email= test@example.com ",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "test@example.com").
					Return([]model.Subscription{
						{ID: 1, RepositoryName: "golang/go", Confirmed: true},
					}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(1), body["total"])
				subs := body["subscriptions"].([]interface{})
				first := subs[0].(map[string]interface{})
				assert.Equal(t, "test@example.com", first["email"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := mocks.NewSubscriptionUseCase(t)
			tt.mockSetup(mockUC)

			r := newRouter(http.MethodGet, "/subscriptions", newTestHandler(mockUC).ListSubscriptions)
			w := doRequest(r, http.MethodGet, "/subscriptions"+tt.query, nil)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkBody != nil {
				var body map[string]interface{}
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tt.checkBody(t, body)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"user+tag@sub.domain.org", false},
		{"", true},
		{"not-an-email", true},
		{"@nodomain.com", true},
		{"user@", true},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := validateEmail(tt.email)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidEmailFormat)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSubscription(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		repo      string
		wantErrs  int
		errSubstr []string
	}{
		{"valid", "user@example.com", "owner/repo", 0, nil},
		{"invalid email only", "bad", "owner/repo", 1, []string{"invalid email"}},
		{"invalid repo only", "user@example.com", "badrepo", 1, []string{"owner/repo"}},
		{"both invalid", "bad", "bad", 2, []string{"invalid email", "owner/repo"}},
		{"repo empty segments", "user@example.com", "/repo", 1, []string{"owner/repo"}},
		{"repo trailing slash", "user@example.com", "owner/", 1, []string{"owner/repo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateSubscription(tt.email, tt.repo)
			assert.Len(t, errs, tt.wantErrs)
			for i, substr := range tt.errSubstr {
				assert.Contains(t, errs[i], substr)
			}
		})
	}
}

func TestHandler_ListSubscriptions_ResponseMapping(t *testing.T) {
	mockUC := mocks.NewSubscriptionUseCase(t)
	email := "user@example.com"

	mockUC.On("ListByEmail", mock.Anything, email).Return([]model.Subscription{
		{ID: 42, RepositoryName: "torvalds/linux", Confirmed: true, LastSeenTag: "v6.9"},
	}, nil).Once()

	r := newRouter(http.MethodGet, "/subscriptions", newTestHandler(mockUC).ListSubscriptions)
	w := doRequest(r, http.MethodGet, "/subscriptions?email="+email, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.ListSubscriptionsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, int64(42), resp.Subscriptions[0].ID)
	assert.Equal(t, email, resp.Subscriptions[0].Email)
	assert.Equal(t, "torvalds/linux", resp.Subscriptions[0].RepositoryName)
	assert.Equal(t, "v6.9", resp.Subscriptions[0].LastSeenTag)
	assert.True(t, resp.Subscriptions[0].Confirmed)

	mockUC.AssertExpectations(t)
}
