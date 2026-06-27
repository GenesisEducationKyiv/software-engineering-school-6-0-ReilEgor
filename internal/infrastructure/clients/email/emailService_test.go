package email

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
)

func cancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

var Errsender = errors.New("smtp: connection refused")

func TestEmailManager_SendConfirmation(t *testing.T) {
	t.Parallel()
	type args struct {
		ctx      context.Context
		to       string
		repoName string
		token    string
	}
	tests := []struct {
		name            string
		baseURL         string
		args            args
		mockReturn      error
		wantErr         bool
		wantErrIs       error
		wantErrContains []string
		wantMsgContains []string
		wantTo          string
	}{
		{
			name:    "success: standard confirmation",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				token:    "conf-123",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"Confirm subscription to owner/repo",
				"owner/repo",
				"http://localhost:8080/api/v1/confirm/conf-123",
			},
		},
		{
			name:    "success: different domain produces correct URL",
			baseURL: "https://reponotifier.com",
			args: args{
				ctx:      context.Background(),
				to:       "admin@test.com",
				repoName: "org/system",
				token:    "token-xyz",
			},
			mockReturn: nil,
			wantTo:     "admin@test.com",
			wantMsgContains: []string{
				"https://reponotifier.com/api/v1/confirm/token-xyz",
				"Confirm subscription to org/system",
			},
		},
		{
			name:    "boundary: empty token produces URL with trailing slash",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				token:    "",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"http://localhost:8080/api/v1/confirm/",
			},
		},
		{
			name:    "boundary: empty repoName",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "",
				token:    "tok",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"Confirm subscription to ",
				"http://localhost:8080/api/v1/confirm/tok",
			},
		},
		{
			name:    "boundary: empty baseURL",
			baseURL: "",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				token:    "tok",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"/api/v1/confirm/tok",
			},
		},
		{
			name:    "error: sender failure is wrapped with op prefix",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				token:    "conf-123",
			},
			mockReturn:      Errsender,
			wantErr:         true,
			wantErrIs:       Errsender,
			wantErrContains: []string{"EmailManager.SendConfirmation", Errsender.Error()},
		},
		{
			name:    "error: cancelled context propagated and wrapped",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      cancelledCtx(),
				to:       "user@test.com",
				repoName: "owner/repo",
				token:    "conf-123",
			},
			mockReturn:      context.Canceled,
			wantErr:         true,
			wantErrIs:       context.Canceled,
			wantErrContains: []string{"EmailManager.SendConfirmation"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSender := mocks.NewEmailSender(t)

			var capturedMsg model.EmailMessage

			mockSender.
				On("Send", mock.Anything, mock.MatchedBy(func(msg model.EmailMessage) bool {
					capturedMsg = msg
					return true
				})).
				Return(tc.mockReturn).
				Once()

			manager := NewEmailManager(mockSender, config.AppBaseURLType(tc.baseURL))
			err := manager.SendConfirmation(tc.args.ctx, tc.args.to, tc.args.repoName, tc.args.token)

			if tc.wantErr {
				require.Error(t, err)

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs,
						"errors.Is must reach root cause through wrapping")
				}
				for _, sub := range tc.wantErrContains {
					assert.Contains(t, err.Error(), sub,
						"error message must contain %q", sub)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantTo, capturedMsg.To,
					"To field must match the caller-supplied address")

				combined := capturedMsg.Subject + capturedMsg.Body
				for _, sub := range tc.wantMsgContains {
					assert.Contains(t, combined, sub,
						"Subject+Body must contain %q", sub)
				}
			}

			mockSender.AssertExpectations(t)
		})
	}
}

func TestEmailManager_SendNotification(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx      context.Context
		to       string
		repoName string
		tag      string
		token    string
	}

	tests := []struct {
		name    string
		baseURL string
		args    args

		mockReturn error

		wantErr         bool
		wantErrIs       error
		wantErrContains []string

		wantMsgContains []string
		wantTo          string
	}{
		{
			name:    "success: new release notification",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "dev@test.com",
				repoName: "golang/go",
				tag:      "v1.22.0",
				token:    "unsub-999",
			},
			mockReturn: nil,
			wantTo:     "dev@test.com",
			wantMsgContains: []string{
				"New release: golang/go",
				"v1.22.0",
				"golang/go",
				"http://localhost:8080/api/v1/unsubscribe/unsub-999",
			},
		},
		{
			name:    "success: pre-release tag contains dots and hyphens",
			baseURL: "https://notify.io",
			args: args{
				ctx:      context.Background(),
				to:       "team@company.com",
				repoName: "company/internal-tool",
				tag:      "2.0.1-beta",
				token:    "tok-beta",
			},
			mockReturn: nil,
			wantTo:     "team@company.com",
			wantMsgContains: []string{
				"company/internal-tool",
				"2.0.1-beta",
				"https://notify.io/api/v1/unsubscribe/tok-beta",
			},
		},
		{
			name:    "boundary: empty tag still produces valid message",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				tag:      "",
				token:    "tok",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"http://localhost:8080/api/v1/unsubscribe/tok",
			},
		},
		{
			name:    "boundary: empty token produces URL with trailing slash",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				tag:      "v1.0.0",
				token:    "",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"http://localhost:8080/api/v1/unsubscribe/",
			},
		},
		{
			name:    "boundary: empty repoName",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "",
				tag:      "v2.0",
				token:    "tok",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"New release: ",
				"v2.0",
			},
		},
		{
			name:    "boundary: empty baseURL produces relative-style link",
			baseURL: "",
			args: args{
				ctx:      context.Background(),
				to:       "user@test.com",
				repoName: "owner/repo",
				tag:      "v1.0",
				token:    "tok",
			},
			mockReturn: nil,
			wantTo:     "user@test.com",
			wantMsgContains: []string{
				"/api/v1/unsubscribe/tok",
			},
		},
		{
			name:    "error: sender failure is wrapped with op prefix",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "dev@test.com",
				repoName: "golang/go",
				tag:      "v1.22.0",
				token:    "unsub-999",
			},
			mockReturn:      Errsender,
			wantErr:         true,
			wantErrIs:       Errsender,
			wantErrContains: []string{"EmailManager.SendNotification", Errsender.Error()},
		},
		{
			name:    "error: cancelled context propagated and wrapped",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      cancelledCtx(),
				to:       "dev@test.com",
				repoName: "golang/go",
				tag:      "v1.22.0",
				token:    "unsub-999",
			},
			mockReturn:      context.Canceled,
			wantErr:         true,
			wantErrIs:       context.Canceled,
			wantErrContains: []string{"EmailManager.SendNotification"},
		},
		{
			name:    "error: custom sentinel survives error wrapping",
			baseURL: "http://localhost:8080",
			args: args{
				ctx:      context.Background(),
				to:       "dev@test.com",
				repoName: "golang/go",
				tag:      "v1.0",
				token:    "tok",
			},
			mockReturn:      fmt.Errorf("quota exceeded: %w", Errsender),
			wantErr:         true,
			wantErrIs:       Errsender,
			wantErrContains: []string{"EmailManager.SendNotification", "quota exceeded"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSender := mocks.NewEmailSender(t)

			var capturedMsg model.EmailMessage

			mockSender.
				On("Send", mock.Anything, mock.MatchedBy(func(msg model.EmailMessage) bool {
					capturedMsg = msg
					return true
				})).
				Return(tc.mockReturn).
				Once()

			manager := NewEmailManager(mockSender, config.AppBaseURLType(tc.baseURL))
			err := manager.SendNotification(
				tc.args.ctx,
				tc.args.to,
				tc.args.repoName,
				tc.args.tag,
				tc.args.token,
			)

			if tc.wantErr {
				require.Error(t, err)

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs,
						"errors.Is must reach root cause through wrapping")
				}
				for _, sub := range tc.wantErrContains {
					assert.Contains(t, err.Error(), sub,
						"error message must contain %q", sub)
				}
			} else {
				require.NoError(t, err)

				assert.Equal(t, tc.wantTo, capturedMsg.To,
					"To field must match the caller-supplied address")

				combined := capturedMsg.Subject + capturedMsg.Body
				for _, sub := range tc.wantMsgContains {
					assert.Contains(t, combined, sub,
						"Subject+Body must contain %q", sub)
				}
			}

			mockSender.AssertExpectations(t)
		})
	}
}
