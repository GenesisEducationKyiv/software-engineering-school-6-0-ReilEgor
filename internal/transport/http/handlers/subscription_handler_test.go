package handlers

import (
	"bytes"
	"context"
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
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/dto"
)

type errorResponse struct {
	Error string `json:"error"`
}

type errorsResponse struct {
	Errors []string `json:"errors"`
}

type messageResponse struct {
	Message string `json:"message"`
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v))
	return v
}

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

func doRequest(r *gin.Engine, method, url string, body any) *httptest.ResponseRecorder {
	var b *bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		b = bytes.NewBuffer(raw)
	} else {
		b = bytes.NewBuffer(nil)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, b)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Subscribe(t *testing.T) {
	tests := []struct {
		name           string
		body           any
		mockSetup      func(uc *mocks.SubscriptionUseCase)
		expectedStatus int
		checkBody      func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "success - 202 accepted",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").Return(nil).Once()
			},
			expectedStatus: http.StatusAccepted,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[messageResponse](t, w)
				assert.Contains(t, resp.Message, "email to confirm")
			},
		},
		{
			name:           "invalid json body",
			body:           "not-json",
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing email",
			body:           map[string]string{"repository": "golang/go"},
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "invalid request body")
			},
		},
		{
			name:           "invalid email format",
			body:           map[string]string{"email": "not-an-email", "repository": "golang/go"},
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "invalid request body")
			},
		},
		{
			name:           "invalid repo format - no slash",
			body:           map[string]string{"email": "test@example.com", "repository": "badrepo"},
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorsResponse](t, w)
				require.NotEmpty(t, resp.Errors)
				assert.Contains(t, resp.Errors[0], "owner/repo")
			},
		},
		{
			name:           "both email and repo invalid - two errors returned",
			body:           map[string]string{"email": "bad", "repository": "bad"},
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "invalid request body")
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
			name: "already subscribed - 409 conflict",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(ErrAlreadySubscribed).Once()
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "github unavailable - 503",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(service.ErrGitHubUnavailable).Once()
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "rate limit exceeded - 503",
			body: map[string]string{"email": "test@example.com", "repository": "golang/go"},
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Subscribe", mock.Anything, "test@example.com", "golang/go").
					Return(service.ErrRateLimitExceeded).Once()
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "unexpected internal error - 500",
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
				tt.checkBody(t, w)
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
		checkBody      func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:  "success - 200",
			token: "valid_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Confirm", mock.Anything, "valid_token").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[messageResponse](t, w)
				assert.Contains(t, resp.Message, "confirmed")
			},
		},
		{
			name:  "invalid token - 404",
			token: "bad_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("Confirm", mock.Anything, "bad_token").Return(model.ErrInvalidToken).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "expired")
			},
		},
		{
			name:  "internal error - 500",
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
				tt.checkBody(t, w)
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
		checkBody      func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:  "success - 200",
			token: "token123",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("UnsubscribeByToken", mock.Anything, "token123").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[messageResponse](t, w)
				assert.Contains(t, resp.Message, "unsubscribed")
			},
		},
		{
			name:  "invalid token - 404",
			token: "expired_token",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("UnsubscribeByToken", mock.Anything, "expired_token").
					Return(model.ErrInvalidToken).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "expired")
			},
		},
		{
			name:  "internal error - 500",
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
				tt.checkBody(t, w)
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
		checkBody      func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:  "success - returns list",
			query: "?email=test@example.com",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "test@example.com").Return([]model.Subscription{
					{ID: 1, RepositoryName: "golang/go", Confirmed: true},
					{ID: 2, RepositoryName: "google/uuid", Confirmed: false},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[dto.ListSubscriptionsResponse](t, w)
				assert.Equal(t, 2, resp.Total)
				assert.Len(t, resp.Subscriptions, 2)
			},
		},
		{
			name:           "missing email - 400",
			query:          "",
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "email")
			},
		},
		{
			name:           "invalid email format - 400",
			query:          "?email=not-valid",
			mockSetup:      func(_ *mocks.SubscriptionUseCase) {},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[errorResponse](t, w)
				assert.Contains(t, resp.Error, "invalid email")
			},
		},
		{
			name:  "empty subscription list - 200 with empty array",
			query: "?email=new@example.com",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "new@example.com").
					Return([]model.Subscription{}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[dto.ListSubscriptionsResponse](t, w)
				assert.Equal(t, 0, resp.Total)
				assert.Empty(t, resp.Subscriptions)
			},
		},
		{
			name:  "usecase error - 500",
			query: "?email=test@example.com",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "test@example.com").
					Return(nil, fmt.Errorf("db connection lost")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:  "email with leading/trailing spaces - trimmed correctly",
			query: "?email= test@example.com ",
			mockSetup: func(uc *mocks.SubscriptionUseCase) {
				uc.On("ListByEmail", mock.Anything, "test@example.com").
					Return([]model.Subscription{
						{ID: 1, RepositoryName: "golang/go", Confirmed: true},
					}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				resp := decodeJSON[dto.ListSubscriptionsResponse](t, w)
				require.Equal(t, 1, resp.Total)
				assert.Equal(t, "test@example.com", resp.Subscriptions[0].Email)
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
				tt.checkBody(t, w)
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
