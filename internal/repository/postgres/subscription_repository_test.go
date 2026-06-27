package postgres

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/model"
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

func TestSubscriptionRepository_GetByToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		mockSetup   func(mock pgxmock.PgxPoolIface, token string)
		expectError bool
		expectErr   error
		checkResult func(t *testing.T, sub *model.Subscription)
	}{
		{
			name:  "success",
			token: "valid-token",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectQuery("^SELECT (.+) FROM subscriptions s").
					WithArgs(token).
					WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "repository_id", "full_name", "token", "is_confirmed", "created_at"}).
						AddRow(int64(1), int64(10), int64(100), "golang/go", token, true, time.Now()))
			},
			expectError: false,
			checkResult: func(t *testing.T, sub *model.Subscription) {
				assert.Equal(t, int64(1), sub.ID)
				assert.Equal(t, "golang/go", sub.RepositoryName)
				assert.True(t, sub.Confirmed)
			},
		},
		{
			name:  "not found error",
			token: "invalid-token",
			mockSetup: func(mock pgxmock.PgxPoolIface, token string) {
				mock.ExpectQuery("^SELECT (.+) FROM subscriptions s").
					WithArgs(token).
					WillReturnError(pgx.ErrNoRows)
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
			sub, err := repo.GetByToken(context.Background(), tt.token)

			if tt.expectError {
				require.Error(t, err)
				if tt.expectErr != nil {
					require.ErrorIs(t, err, tt.expectErr)
				}
				assert.Nil(t, sub)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, sub)
				tt.checkResult(t, sub)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_GetByRepoID(t *testing.T) {
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
				mock.ExpectQuery("^SELECT u.email, s.token").
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
				mock.ExpectQuery("^SELECT u.email, s.token").
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
				mock.ExpectQuery("^SELECT u.email, s.token").
					WithArgs(repoID).
					WillReturnError(fmt.Errorf("db fail"))
			},
			expectError: true,
			checkErrMsg: "query:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock)
			subs, err := repo.GetByRepoID(context.Background(), repoID)

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
	cols := []string{"id", "repository_id", "full_name", "token", "is_confirmed", "last_seen_tag", "created_at"}

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
						AddRow(int64(1), int64(101), "golang/go", "token1", true, "v1.25.0", now).
						AddRow(int64(2), int64(102), "google/uuid", "token2", false, "v1.6.0", now))
			},
			checkResult: func(t *testing.T, subs []model.Subscription) {
				assert.Len(t, subs, 2)
				assert.Equal(t, "golang/go", subs[0].RepositoryName)
				assert.True(t, subs[0].Confirmed)
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
				assert.Empty(t, subs)
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

func TestSubscriptionRepository_Save(t *testing.T) {
	tests := []struct {
		name        string
		inputSub    *model.Subscription
		mockSetup   func(mock pgxmock.PgxPoolIface, sub *model.Subscription)
		expectError bool
	}{
		{
			name: "success save",
			inputSub: &model.Subscription{
				UserID:       10,
				RepositoryID: 20,
				Token:        "token-123",
				Confirmed:    true,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, sub *model.Subscription) {
				mock.ExpectQuery("^INSERT INTO subscriptions").
					WithArgs(sub.UserID, sub.RepositoryID, sub.Token, sub.Confirmed).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(99)))
			},
			expectError: false,
		},
		{
			name: "save error",
			inputSub: &model.Subscription{
				UserID:       10,
				RepositoryID: 20,
				Token:        "token-123",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, sub *model.Subscription) {
				mock.ExpectQuery("^INSERT INTO subscriptions").
					WithArgs(sub.UserID, sub.RepositoryID, sub.Token, sub.Confirmed).
					WillReturnError(fmt.Errorf("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newSubRepo(t)
			defer mock.Close()

			tt.mockSetup(mock, tt.inputSub)
			err := repo.Save(context.Background(), tt.inputSub)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(99), tt.inputSub.ID)
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
				mock.ExpectExec("^DELETE FROM subscriptions").
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
				mock.ExpectExec("^DELETE FROM subscriptions").
					WithArgs(userID, repo).
					WillReturnError(fmt.Errorf("fatal error"))
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
