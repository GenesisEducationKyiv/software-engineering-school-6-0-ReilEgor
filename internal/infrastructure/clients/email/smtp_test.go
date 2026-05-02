package email

import (
	"context"
	"errors"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ReilEgor/RepoNotifier/internal/config"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
)

func newTestClient(sendMail func(string, smtp.Auth, string, []string, []byte) error) *SmtpClient {
	c := NewSmtpClient(
		"localhost",
		"25",
		"from@example.com",
		"pass",
		"user",
		"http://localhost:8080",
	)
	c.sendMail = sendMail
	return c
}

func stubbedSendMail(err error) func(string, smtp.Auth, string, []string, []byte) error {
	return func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
		return err
	}
}

func TestSmtpClient_SendNotification(t *testing.T) {
	tests := []struct {
		name        string
		to          string
		repoName    string
		tagName     string
		token       string
		sendMailErr error
		wantErr     error
	}{
		{
			name:        "success",
			to:          "user@example.com",
			repoName:    "owner/repo",
			tagName:     "v1.0.0",
			token:       "test-token",
			sendMailErr: nil,
			wantErr:     nil,
		},
		{
			name:        "smtp auth failed — code 535",
			to:          "user@example.com",
			repoName:    "owner/repo",
			tagName:     "v1.0.0",
			token:       "test-token",
			sendMailErr: errors.New("535 5.7.8 Authentication failed"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "smtp auth failed — text match",
			to:          "user@example.com",
			repoName:    "owner/repo",
			tagName:     "v1.0.0",
			token:       "test-token",
			sendMailErr: errors.New("Authentication failed: bad credentials"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "smtp server unavailable — connection refused",
			to:          "user@example.com",
			repoName:    "owner/repo",
			tagName:     "v1.0.0",
			token:       "test-token",
			sendMailErr: errors.New("connection refused"),
			wantErr:     service.ErrSMTPUnavailable,
		},
		{
			name:        "smtp server unavailable — timeout",
			to:          "user@example.com",
			repoName:    "owner/repo",
			tagName:     "v1.0.0",
			token:       "test-token",
			sendMailErr: errors.New("dial tcp: i/o timeout"),
			wantErr:     service.ErrSMTPUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(stubbedSendMail(tt.sendMailErr))
			err := client.SendNotification(context.Background(), tt.to, tt.repoName, tt.tagName, tt.token)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestSmtpClient_SendConfirmation(t *testing.T) {
	tests := []struct {
		name        string
		to          string
		repoName    string
		token       string
		sendMailErr error
		wantErr     error
	}{
		{
			name:        "success",
			to:          "user@example.com",
			repoName:    "owner/repo",
			token:       "confirm-token",
			sendMailErr: nil,
		},
		{
			name:        "auth failed",
			to:          "user@example.com",
			repoName:    "owner/repo",
			token:       "confirm-token",
			sendMailErr: errors.New("535 Authentication failed"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "server unavailable",
			to:          "user@example.com",
			repoName:    "owner/repo",
			token:       "confirm-token",
			sendMailErr: errors.New("connection refused"),
			wantErr:     service.ErrSMTPUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(stubbedSendMail(tt.sendMailErr))
			err := client.SendConfirmation(context.Background(), tt.to, tt.repoName, tt.token)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestClassifySmtpError(t *testing.T) {
	tests := []struct {
		errMsg  string
		wantErr error
	}{
		{"535 5.7.8 Authentication failed", service.ErrAuthFailed},
		{"Authentication failed: invalid credentials", service.ErrAuthFailed},
		{"AUTHENTICATION FAILED", service.ErrAuthFailed},
		{"connection refused", service.ErrSMTPUnavailable},
		{"dial tcp: i/o timeout", service.ErrSMTPUnavailable},
		{"unexpected EOF", service.ErrSMTPUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			got := classifySmtpError(errors.New(tt.errMsg))
			assert.ErrorIs(t, got, tt.wantErr)
		})
	}
}

func TestSmtpClient_buildMessage(t *testing.T) {
	const baseURL = "http://localhost:8080"

	tests := []struct {
		name          string
		to            string
		repoName      string
		tagName       string
		token         string
		expectContain []string
	}{
		{
			name:     "contains headers and release info",
			to:       "user@example.com",
			repoName: "owner/repo",
			tagName:  "v2.0.0",
			token:    "secret-abc",
			expectContain: []string{
				"From: from@example.com",
				"To: user@example.com",
				"Subject: New release in owner/repo!",
				"A new version v2.0.0 has been released",
			},
		},
		{
			name:     "contains github release url",
			to:       "user@example.com",
			repoName: "owner/repo",
			tagName:  "v3.0.0",
			token:    "secret-abc",
			expectContain: []string{
				"https://github.com/owner/repo/releases/tag/v3.0.0",
			},
		},
		{
			name:     "contains unsubscribe link with token",
			to:       "user@example.com",
			repoName: "owner/repo",
			tagName:  "v1.0.0",
			token:    "unsub-token-xyz",
			expectContain: []string{
				"http://localhost:8080/api/v1/unsubscribe/unsub-token-xyz",
			},
		},
		{
			name:     "content type is plain text utf-8",
			to:       "user@example.com",
			repoName: "owner/repo",
			tagName:  "v1.0.0",
			token:    "tok",
			expectContain: []string{
				"Content-Type: text/plain; charset=UTF-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSmtpClient("localhost", "25", "from@example.com", "pass", "user", config.AppBaseURLType(baseURL))
			msg := string(c.buildMessage(tt.to, tt.repoName, tt.tagName, tt.token))

			for _, expected := range tt.expectContain {
				assert.Contains(t, msg, expected)
			}
		})
	}
}

func TestSmtpClient_buildConfirmMessage(t *testing.T) {
	const baseURL = "http://localhost:8080"

	tests := []struct {
		name          string
		to            string
		repoName      string
		token         string
		expectContain []string
	}{
		{
			name:     "contains confirm subject and link",
			to:       "user@example.com",
			repoName: "owner/repo",
			token:    "confirm-xyz",
			expectContain: []string{
				"Subject: Confirm your subscription to owner/repo",
				"http://localhost:8080/api/v1/confirm/confirm-xyz",
				"From: from@example.com",
				"To: user@example.com",
			},
		},
		{
			name:     "contains repo name in body",
			to:       "user@example.com",
			repoName: "torvalds/linux",
			token:    "tok",
			expectContain: []string{
				"torvalds/linux",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSmtpClient("localhost", "25", "from@example.com", "pass", "user", config.AppBaseURLType(baseURL))
			msg := string(c.buildConfirmMessage(tt.to, tt.repoName, tt.token))

			for _, expected := range tt.expectContain {
				assert.Contains(t, msg, expected)
			}
		})
	}
}
