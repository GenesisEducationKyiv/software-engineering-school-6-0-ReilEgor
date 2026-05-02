package postgres

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
)

func newSubRepo(t *testing.T) (*SubscriptionRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	repo := &SubscriptionRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return repo, mock
}

func TestSubscriptionRepository_Confirm(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		mockSetup   func(mock pgxmock.PgxPoolIface, token string)
		expectError bool
		expectErr   error
		checkErrMsg string
	}{
		{
			name:  "success confirm",
			token: "valid-token",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("UPDATE subscriptions SET is_confirmed = TRUE").
					WithArgs(token).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectError: false,
		},
		{
			name:  "invalid token - no rows affected",
			token: "wrong-token",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("UPDATE subscriptions").
					WithArgs(token).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			expectError: true,
			expectErr:   model.ErrInvalidToken,
		},
		{
			name:  "database exec error",
			token: "any-token",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("UPDATE subscriptions").
					WithArgs(token).
					WillReturnError(fmt.Errorf("connection reset"))
			},
			expectError: true,
			checkErrMsg: "exec:",
		},
		{
			name:  "empty token",
			token: "",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("UPDATE subscriptions").
					WithArgs(token).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			expectError: true,
			expectErr:   model.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock, tt.token)
			err := repo.Confirm(context.Background(), tt.token)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectErr != nil {
					require.ErrorIs(t, err, tt.expectErr)
				}
				if tt.checkErrMsg != "" {
					assert.Contains(t, err.Error(), tt.checkErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_UnsubscribeByToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		mockSetup   func(mock pgxmock.PgxPoolIface, token string)
		expectError bool
		expectErr   error
		checkErrMsg string
	}{
		{
			name:  "success unsubscribe",
			token: "unsubscribe-me",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("DELETE FROM subscriptions WHERE token = \\$1").
					WithArgs(token).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			expectError: false,
		},
		{
			name:  "invalid token - no rows deleted",
			token: "fake",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(token).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			expectError: true,
			expectErr:   model.ErrInvalidToken,
		},
		{
			name:  "database exec error",
			token: "any-token",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(token).
					WillReturnError(fmt.Errorf("db unavailable"))
			},
			expectError: true,
			checkErrMsg: "exec:",
		},
		{
			name:  "empty token",
			token: "",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(token).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			expectError: true,
			expectErr:   model.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock, tt.token)
			err := repo.UnsubscribeByToken(context.Background(), tt.token)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectErr != nil {
					require.ErrorIs(t, err, tt.expectErr)
				}
				if tt.checkErrMsg != "" {
					assert.Contains(t, err.Error(), tt.checkErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_GetSubscribersByRepoID(t *testing.T) {
	repoID := int64(42)

	tests := []struct {
		name        string
		mockSetup   func(mock pgxmock.PgxPoolIface)
		expectError bool
		checkErrMsg string
		checkResult func(t *testing.T, subs []model.Subscriber)
	}{
		{
			name: "success with multiple subscribers",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT u.email, s.token FROM subscriptions s").
					WithArgs(repoID).
					WillReturnRows(pgxmock.NewRows([]string{"email", "token"}).
						AddRow("user1@mail.com", "token1").
						AddRow("user2@mail.com", "token2"))
			},
			checkResult: func(t *testing.T, subs []model.Subscriber) {
				assert.Len(t, subs, 2)
				assert.Equal(t, "user1@mail.com", subs[0].Email)
				assert.Equal(t, "token1", subs[0].Token)
				assert.Equal(t, "user2@mail.com", subs[1].Email)
				assert.Equal(t, "token2", subs[1].Token)
			},
		},
		{
			name: "empty result - no confirmed subscribers",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT u.email, s.token FROM subscriptions s").
					WithArgs(repoID).
					WillReturnRows(pgxmock.NewRows([]string{"email", "token"}))
			},
			checkResult: func(t *testing.T, subs []model.Subscriber) {
				assert.Nil(t, subs)
			},
		},
		{
			name: "query error",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT").
					WithArgs(repoID).
					WillReturnError(fmt.Errorf("db fail"))
			},
			expectError: true,
			checkErrMsg: "query:",
		},
		{
			name: "row scan error - wrong email type",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT u.email, s.token FROM subscriptions s").
					WithArgs(repoID).
					WillReturnRows(pgxmock.NewRows([]string{"email", "token"}).
						AddRow(12345, "token1"))
			},
			expectError: true,
			checkErrMsg: "scan:",
		},
		{
			name: "single subscriber",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT u.email, s.token FROM subscriptions s").
					WithArgs(repoID).
					WillReturnRows(pgxmock.NewRows([]string{"email", "token"}).
						AddRow("only@one.com", "solo-token"))
			},
			checkResult: func(t *testing.T, subs []model.Subscriber) {
				assert.Len(t, subs, 1)
				assert.Equal(t, "only@one.com", subs[0].Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock)
			subs, err := repo.GetSubscribersByRepoID(context.Background(), repoID)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
				assert.Nil(t, subs)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, subs)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_GetByEmail(t *testing.T) {
	email := "yehor@kpi.ua"
	now := time.Now()
	cols := []string{"id", "repository_id", "full_name", "is_confirmed", "last_seen_tag", "created_at"}

	tests := []struct {
		name        string
		mockSetup   func(mock pgxmock.PgxPoolIface)
		expectError bool
		checkErrMsg string
		checkResult func(t *testing.T, subs []model.Subscription)
	}{
		{
			name: "success with multiple subscriptions",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM subscriptions s").
					WithArgs(email).
					WillReturnRows(pgxmock.NewRows(cols).
						AddRow(int64(1), int64(101), "golang/go", true, "v1.25.0", now).
						AddRow(int64(2), int64(102), "google/uuid", false, "v1.6.0", now))
			},
			checkResult: func(t *testing.T, subs []model.Subscription) {
				assert.Len(t, subs, 2)
				assert.Equal(t, "golang/go", subs[0].RepositoryName)
				assert.True(t, subs[0].Confirmed)
				assert.Equal(t, int64(101), subs[0].RepositoryID)
				assert.Equal(t, "google/uuid", subs[1].RepositoryName)
				assert.False(t, subs[1].Confirmed)
			},
		},
		{
			name: "empty result - returns empty slice, not nil",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM subscriptions s").
					WithArgs(email).
					WillReturnRows(pgxmock.NewRows(cols))
			},
			checkResult: func(t *testing.T, subs []model.Subscription) {
				assert.NotNil(t, subs)
				assert.Empty(t, subs, 0)
			},
		},
		{
			name: "query error",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM subscriptions s").
					WithArgs(email).
					WillReturnError(fmt.Errorf("timeout"))
			},
			expectError: true,
		},
		{
			name: "scan error - wrong id type",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM subscriptions s").
					WithArgs(email).
					WillReturnRows(pgxmock.NewRows(cols).
						AddRow("not-int", int64(101), "golang/go", true, "v1.25.0", now))
			},
			expectError: true,
			checkErrMsg: "scan:",
		},
		{
			name: "rows iteration error",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows(cols).
					AddRow(int64(1), int64(101), "golang/go", true, "v1.25.0", now).
					RowError(0, fmt.Errorf("network error during iteration"))
				mock.ExpectQuery("SELECT (.+) FROM subscriptions s").
					WithArgs(email).
					WillReturnRows(rows)
			},
			expectError: true,
			checkErrMsg: "scan:",
		},
		{
			name: "single confirmed subscription",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM subscriptions s").
					WithArgs(email).
					WillReturnRows(pgxmock.NewRows(cols).
						AddRow(int64(7), int64(55), "torvalds/linux", true, "v6.9", now))
			},
			checkResult: func(t *testing.T, subs []model.Subscription) {
				assert.Len(t, subs, 1)
				assert.Equal(t, int64(7), subs[0].ID)
				assert.Equal(t, "torvalds/linux", subs[0].RepositoryName)
				assert.True(t, subs[0].Confirmed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock)
			result, err := repo.GetByEmail(context.Background(), email)

			if tt.expectError {
				require.Error(t, err)
				if tt.checkErrMsg != "" {
					assert.Contains(t, err.Error(), tt.checkErrMsg)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_Delete(t *testing.T) {
	userID := int64(1)
	repoName := "owner/repo"

	tests := []struct {
		name        string
		userID      int64
		repoName    string
		mockSetup   func(mock pgxmock.PgxPoolIface, userID int64, repo string)
		expectError bool
		checkErrMsg string
	}{
		{
			name:     "success delete",
			userID:   userID,
			repoName: repoName,
			mockSetup: func(mock pgxmock.PgxPoolIface, userID int64, repo string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(userID, repo).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			expectError: false,
		},
		{
			name:     "exec error",
			userID:   userID,
			repoName: repoName,
			mockSetup: func(mock pgxmock.PgxPoolIface, userID int64, repo string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(userID, repo).
					WillReturnError(fmt.Errorf("fatal error"))
			},
			expectError: true,
			checkErrMsg: "exec:",
		},
		{
			name:     "zero rows affected - subscription did not exist",
			userID:   userID,
			repoName: "nonexistent/repo",
			mockSetup: func(mock pgxmock.PgxPoolIface, userID int64, repo string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(userID, repo).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			expectError: false,
		},
		{
			name:     "zero user id",
			userID:   0,
			repoName: repoName,
			mockSetup: func(mock pgxmock.PgxPoolIface, userID int64, repo string) {
				mock.ExpectExec("DELETE FROM subscriptions").
					WithArgs(userID, repo).
					WillReturnError(fmt.Errorf("not null violation"))
			},
			expectError: true,
			checkErrMsg: "exec:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock, tt.userID, tt.repoName)
			err := repo.Delete(context.Background(), tt.userID, tt.repoName)

			if tt.expectError {
				require.Error(t, err)
				if tt.checkErrMsg != "" {
					assert.Contains(t, err.Error(), tt.checkErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
