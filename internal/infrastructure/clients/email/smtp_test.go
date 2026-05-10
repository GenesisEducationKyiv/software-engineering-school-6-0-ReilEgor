package email

import (
	"context"
	"errors"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/service"
)

func smtpCancelledCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

type capturedCall struct {
	addr string
	from string
	to   []string
	msg  []byte
}

func makeFakeSendMail(out *capturedCall, returnErr error) func(string, smtp.Auth, string, []string, []byte) error {
	return func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		out.addr = addr
		out.from = from
		out.to = to
		out.msg = msg
		return returnErr
	}
}

func TestSMTPClient_Send(t *testing.T) {
	t.Parallel()

	const (
		host = "smtp.example.com"
		port = "587"
		from = "noreply@example.com"
		pass = "s3cr3t"
		user = "noreply@example.com"
	)

	defaultMsg := model.EmailMessage{
		To:      "user@test.com",
		Subject: "Test Subject",
		Body:    "Hello, world!",
	}

	tests := []struct {
		name               string
		ctx                context.Context
		host               string
		port               string
		from               string
		msg                model.EmailMessage
		sendMailErr        error
		wantErr            error
		wantNoErr          bool
		wantAddrContains   string
		wantRawContains    []string
		wantRawNotContains []string
		wantRecipients     []string
		wantFromArg        string
	}{
		{
			name:             "success: standard message",
			ctx:              context.Background(),
			msg:              defaultMsg,
			wantNoErr:        true,
			wantAddrContains: host + ":" + port,
			wantRawContains: []string{
				"From: " + from,
				"To: " + defaultMsg.To,
				"Subject: " + defaultMsg.Subject,
				"Content-Type: text/plain; charset=UTF-8",
				defaultMsg.Body,
				"\r\n",
			},
			wantRecipients: []string{defaultMsg.To},
			wantFromArg:    from,
		},
		{
			name: "success: multiline body preserved verbatim",
			ctx:  context.Background(),
			msg: model.EmailMessage{
				To:      "a@b.com",
				Subject: "Multiline",
				Body:    "Line1\nLine2\nLine3",
			},
			wantNoErr: true,
			wantRawContains: []string{
				"Line1\nLine2\nLine3",
			},
			wantRecipients: []string{"a@b.com"},
			wantFromArg:    from,
		},
		{
			name: "boundary: empty subject",
			ctx:  context.Background(),
			msg: model.EmailMessage{
				To:      "user@test.com",
				Subject: "",
				Body:    "body text",
			},
			wantNoErr: true,
			wantRawContains: []string{
				"Subject: \r\n",
				"body text",
			},
			wantRecipients: []string{"user@test.com"},
		},
		{
			name: "boundary: empty body",
			ctx:  context.Background(),
			msg: model.EmailMessage{
				To:      "user@test.com",
				Subject: "Hi",
				Body:    "",
			},
			wantNoErr: true,
			wantRawContains: []string{
				"Subject: Hi",
			},
			wantRecipients: []string{"user@test.com"},
		},
		{
			name: "boundary: unicode characters in body",
			ctx:  context.Background(),
			msg: model.EmailMessage{
				To:      "user@test.com",
				Subject: "Підтвердження",
				Body:    "Привіт!\n\nПідтвердіть підписку.",
			},
			wantNoErr: true,
			wantRawContains: []string{
				"Підтвердження",
				"Привіт!",
			},
			wantRecipients: []string{"user@test.com"},
		},
		{
			name:             "boundary: custom host and port reflected in addr",
			ctx:              context.Background(),
			host:             "mail.custom.io",
			port:             "2525",
			msg:              defaultMsg,
			wantNoErr:        true,
			wantAddrContains: "mail.custom.io:2525",
			wantRecipients:   []string{defaultMsg.To},
		},
		{
			name:      "boundary: custom from address reflected in header and envelope",
			ctx:       context.Background(),
			from:      "custom@sender.io",
			msg:       defaultMsg,
			wantNoErr: true,
			wantRawContains: []string{
				"From: custom@sender.io",
			},
			wantFromArg: "custom@sender.io",
		},
		{
			name:        "error: 535 code maps to ErrAuthFailed",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("535 Authentication failed"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "error: mixed-case authentication keyword maps to ErrAuthFailed",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("authentication Failed"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "error: lowercase authentication failed maps to ErrAuthFailed",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("authentication failed"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "error: 535 embedded in longer error message maps to ErrAuthFailed",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("smtp: 535 5.7.8 Username and Password not accepted"),
			wantErr:     service.ErrAuthFailed,
		},
		{
			name:        "error: connection refused maps to ErrSMTPUnavailable",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("connection refused"),
			wantErr:     service.ErrSMTPUnavailable,
		},
		{
			name:        "error: dial timeout maps to ErrSMTPUnavailable",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("dial tcp: i/o timeout"),
			wantErr:     service.ErrSMTPUnavailable,
		},
		{
			name:        "error: generic one-word error maps to ErrSMTPUnavailable",
			ctx:         context.Background(),
			msg:         defaultMsg,
			sendMailErr: errors.New("EOF"),
			wantErr:     service.ErrSMTPUnavailable,
		},
		{
			name:        "error: cancelled context error classified as ErrSMTPUnavailable",
			ctx:         smtpCancelledCtx(),
			msg:         defaultMsg,
			sendMailErr: context.Canceled,
			wantErr:     service.ErrSMTPUnavailable,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resolvedHost := host
			if tc.host != "" {
				resolvedHost = tc.host
			}
			resolvedPort := port
			if tc.port != "" {
				resolvedPort = tc.port
			}
			resolvedFrom := from
			if tc.from != "" {
				resolvedFrom = tc.from
			}

			client := NewSMTPClient(
				config.EmailHostType(resolvedHost),
				config.EmailPortType(resolvedPort),
				config.EmailFromType(resolvedFrom),
				pass,
				user,
			)

			var captured capturedCall
			client.sendMail = makeFakeSendMail(&captured, tc.sendMailErr)

			err := client.Send(tc.ctx, tc.msg)
			if tc.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.wantErr,
					"errors.Is must identify the classified sentinel")
				if tc.sendMailErr != nil {
					require.NotErrorIs(t, err, tc.sendMailErr,
						"raw sendMail error must not be returned directly")
				}
				return
			}

			require.NoError(t, err)
			rawStr := string(captured.msg)

			if tc.wantAddrContains != "" {
				assert.Contains(t, captured.addr, tc.wantAddrContains,
					"addr argument must embed host:port")
			}
			if tc.wantFromArg != "" {
				assert.Equal(t, tc.wantFromArg, captured.from,
					"SMTP envelope from must match configured address")
			}
			if tc.wantRecipients != nil {
				assert.Equal(t, tc.wantRecipients, captured.to,
					"SMTP recipient list must contain exactly the message.To address")
			}
			for _, sub := range tc.wantRawContains {
				assert.Contains(t, rawStr, sub, "raw SMTP message must contain %q", sub)
			}

			for _, sub := range tc.wantRawNotContains {
				assert.NotContains(t, rawStr, sub, "raw SMTP message must NOT contain %q", sub)
			}
		})
	}
}

func TestClassifySMTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		errMsg  string
		wantErr error
	}{
		{name: "535 alone", errMsg: "535", wantErr: service.ErrAuthFailed},
		{
			name:    "535 with description",
			errMsg:  "535 Authentication credentials invalid",
			wantErr: service.ErrAuthFailed,
		},
		{name: "535 embedded mid-message", errMsg: "smtp server said: 535 bad auth", wantErr: service.ErrAuthFailed},

		{name: "authentication failed lowercase", errMsg: "authentication failed", wantErr: service.ErrAuthFailed},
		{name: "Authentication Failed title-case", errMsg: "Authentication Failed", wantErr: service.ErrAuthFailed},
		{name: "AUTHENTICATION FAILED uppercase", errMsg: "AUTHENTICATION FAILED", wantErr: service.ErrAuthFailed},
		{
			name:    "authentication failed embedded",
			errMsg:  "smtp: 534 authentication failed please try again",
			wantErr: service.ErrAuthFailed,
		},
		{name: "connection refused", errMsg: "connection refused", wantErr: service.ErrSMTPUnavailable},
		{name: "i/o timeout", errMsg: "dial tcp: i/o timeout", wantErr: service.ErrSMTPUnavailable},
		{name: "EOF", errMsg: "EOF", wantErr: service.ErrSMTPUnavailable},
		{name: "TLS error", errMsg: "tls: handshake failure", wantErr: service.ErrSMTPUnavailable},
		{name: "empty string", errMsg: "", wantErr: service.ErrSMTPUnavailable},
		{name: "authentication without failed", errMsg: "authentication required", wantErr: service.ErrSMTPUnavailable},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := classifySMTPError(errors.New(tc.errMsg))
			assert.ErrorIs(t, got, tc.wantErr,
				"classifySMTPError(%q) = %v, want %v", tc.errMsg, got, tc.wantErr)
		})
	}
}
