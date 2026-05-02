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

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
)

func TestRepositoryRepository_GetAll(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &RepositoryRepository{
		db:     mock,
		logger: discardLogger,
	}

	now := time.Now()

	tests := []struct {
		name          string
		mockSetup     func()
		expectError   bool
		checkErrMsg   string
		expectedLen   int
		expectedRepos []model.Repository
	}{
		{
			name: "success with active repos",
			mockSetup: func() {
				mock.ExpectQuery("SELECT (.+) FROM repositories r WHERE EXISTS").
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
						AddRow(int64(1), "golang/go", "v1.25.0", now).
						AddRow(int64(2), "google/wire", "v0.6.0", now))
			},
			expectError: false,
			expectedLen: 2,
			expectedRepos: []model.Repository{
				{ID: 1, FullName: "golang/go", LastSeenTag: "v1.25.0", UpdatedAt: now},
				{ID: 2, FullName: "google/wire", LastSeenTag: "v0.6.0", UpdatedAt: now},
			},
		},
		{
			name: "success empty result",
			mockSetup: func() {
				mock.ExpectQuery("SELECT (.+) FROM repositories r WHERE EXISTS").
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}))
			},
			expectError: false,
			expectedLen: 0,
		},
		{
			name: "database query error",
			mockSetup: func() {
				mock.ExpectQuery("SELECT (.+) FROM repositories").
					WillReturnError(fmt.Errorf("db connection lost"))
			},
			expectError: true,
			checkErrMsg: "query:",
		},
		{
			name: "row scan error — wrong id type",
			mockSetup: func() {
				mock.ExpectQuery("SELECT (.+) FROM repositories").
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
						AddRow("not-an-id", "owner/repo", "v1", now))
			},
			expectError: true,
			checkErrMsg: "scan:",
		},
		{
			name: "rows iteration error",
			mockSetup: func() {
				rows := pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
					AddRow(int64(1), "golang/go", "v1.25.0", now).
					RowError(0, fmt.Errorf("network failure during iteration"))
				mock.ExpectQuery("SELECT (.+) FROM repositories").
					WillReturnRows(rows)
			},
			expectError: true,
			checkErrMsg: "scan:",
		},
		{
			name: "success with single repo",
			mockSetup: func() {
				mock.ExpectQuery("SELECT (.+) FROM repositories r WHERE EXISTS").
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
						AddRow(int64(42), "torvalds/linux", "v6.9", now))
			},
			expectError: false,
			expectedLen: 1,
			expectedRepos: []model.Repository{
				{ID: 42, FullName: "torvalds/linux", LastSeenTag: "v6.9", UpdatedAt: now},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			result, err := repo.GetAll(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
				for i, expected := range tt.expectedRepos {
					assert.Equal(t, expected.ID, result[i].ID)
					assert.Equal(t, expected.FullName, result[i].FullName)
					assert.Equal(t, expected.LastSeenTag, result[i].LastSeenTag)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepositoryRepository_UpdateLastSeenTag(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	repo := &RepositoryRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name        string
		repoName    string
		tag         string
		mockSetup   func(name, tag string)
		expectError bool
		checkErrMsg string
	}{
		{
			name:     "success update",
			repoName: "golang/go",
			tag:      "v1.26.0",
			mockSetup: func(name, tag string) {
				mock.ExpectExec("UPDATE repositories SET last_seen_tag = \\$1").
					WithArgs(tag, name).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectError: false,
		},
		{
			name:     "database exec error",
			repoName: "golang/go",
			tag:      "v1.26.0",
			mockSetup: func(name, tag string) {
				mock.ExpectExec("UPDATE repositories SET last_seen_tag = \\$1").
					WithArgs(tag, name).
					WillReturnError(fmt.Errorf("connection refused"))
			},
			expectError: true,
			checkErrMsg: "exec:",
		},
		{
			name:     "update non-existent repo — zero rows affected",
			repoName: "nonexistent/repo",
			tag:      "v0.1.0",
			mockSetup: func(name, tag string) {
				mock.ExpectExec("UPDATE repositories SET last_seen_tag = \\$1").
					WithArgs(tag, name).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			expectError: false,
		},
		{
			name:     "empty tag string",
			repoName: "golang/go",
			tag:      "",
			mockSetup: func(name, tag string) {
				mock.ExpectExec("UPDATE repositories SET last_seen_tag = \\$1").
					WithArgs(tag, name).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.repoName, tt.tag)
			err := repo.UpdateLastSeenTag(context.Background(), tt.repoName, tt.tag)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepositoryRepository_GetOrCreate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	repo := &RepositoryRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	now := time.Now()

	tests := []struct {
		name        string
		repoName    string
		tagName     string
		mockSetup   func(name, tag string)
		expectError bool
		checkErrMsg string
		checkResult func(t *testing.T, r *model.Repository)
	}{
		{
			name:     "success upsert — new repo",
			repoName: "golang/go",
			tagName:  "v1.25.0",
			mockSetup: func(name, tag string) {
				mock.ExpectQuery("INSERT INTO repositories (.+) ON CONFLICT").
					WithArgs(name, tag).
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
						AddRow(int64(1), name, tag, now))
			},
			expectError: false,
			checkResult: func(t *testing.T, r *model.Repository) {
				assert.Equal(t, int64(1), r.ID)
				assert.Equal(t, "golang/go", r.FullName)
				assert.Equal(t, "v1.25.0", r.LastSeenTag)
			},
		},
		{
			name:     "success upsert — existing repo returns same id",
			repoName: "golang/go",
			tagName:  "v1.26.0",
			mockSetup: func(name, tag string) {
				mock.ExpectQuery("INSERT INTO repositories (.+) ON CONFLICT").
					WithArgs(name, tag).
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
						AddRow(int64(1), name, name, now))
			},
			expectError: false,
			checkResult: func(t *testing.T, r *model.Repository) {
				assert.Equal(t, int64(1), r.ID)
				assert.NotNil(t, r)
			},
		},
		{
			name:     "database error on insert",
			repoName: "golang/go",
			tagName:  "v1.25.0",
			mockSetup: func(name, tag string) {
				mock.ExpectQuery("INSERT INTO repositories (.+) ON CONFLICT").
					WithArgs(name, tag).
					WillReturnError(fmt.Errorf("deadlock detected"))
			},
			expectError: true,
			checkErrMsg: "query row:",
		},
		{
			name:     "scan error — unexpected column type",
			repoName: "golang/go",
			tagName:  "v1.25.0",
			mockSetup: func(name, tag string) {
				mock.ExpectQuery("INSERT INTO repositories (.+) ON CONFLICT").
					WithArgs(name, tag).
					WillReturnError(fmt.Errorf("scan error: destination id has incompatible type"))
			},
			expectError: true,
			checkErrMsg: "query row:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.repoName, tt.tagName)
			result, err := repo.GetOrCreate(context.Background(), tt.repoName, tt.tagName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSubscriptionRepository_CreatePending(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	repo := &SubscriptionRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name        string
		userID      int64
		repoID      int64
		token       string
		mockSetup   func(userID, repoID int64, token string)
		expectError bool
		expectedID  int64
	}{
		{
			name:   "success create pending",
			userID: 10,
			repoID: 20,
			token:  "secure-token",
			mockSetup: func(userID, repoID int64, token string) {
				mock.ExpectQuery("INSERT INTO subscriptions").
					WithArgs(userID, repoID, token).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(100)))
			},
			expectError: false,
			expectedID:  100,
		},
		{
			name:   "unique constraint violation",
			userID: 10,
			repoID: 20,
			token:  "secure-token",
			mockSetup: func(userID, repoID int64, token string) {
				mock.ExpectQuery("INSERT INTO subscriptions").
					WithArgs(userID, repoID, token).
					WillReturnError(fmt.Errorf("unique constraint"))
			},
			expectError: true,
			expectedID:  0,
		},
		{
			name:   "database connection error",
			userID: 10,
			repoID: 20,
			token:  "token",
			mockSetup: func(userID, repoID int64, token string) {
				mock.ExpectQuery("INSERT INTO subscriptions").
					WithArgs(userID, repoID, token).
					WillReturnError(fmt.Errorf("connection reset by peer"))
			},
			expectError: true,
			expectedID:  0,
		},
		{
			name:   "re-subscribe — ON CONFLICT updates token, returns existing id",
			userID: 5,
			repoID: 99,
			token:  "new-token",
			mockSetup: func(userID, repoID int64, token string) {
				mock.ExpectQuery("INSERT INTO subscriptions").
					WithArgs(userID, repoID, token).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(77)))
			},
			expectError: false,
			expectedID:  77,
		},
		{
			name:   "zero user id",
			userID: 0,
			repoID: 1,
			token:  "token",
			mockSetup: func(userID, repoID int64, token string) {
				mock.ExpectQuery("INSERT INTO subscriptions").
					WithArgs(userID, repoID, token).
					WillReturnError(fmt.Errorf("not null violation"))
			},
			expectError: true,
			expectedID:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.userID, tt.repoID, tt.token)
			id, err := repo.CreatePending(context.Background(), tt.userID, tt.repoID, tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, int64(0), id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
