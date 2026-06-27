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

func TestRepositoryRepository_GetAll(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
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
				mock.ExpectQuery("^SELECT (.+) FROM repositories r WHERE EXISTS").
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
				mock.ExpectQuery("^SELECT (.+) FROM repositories r WHERE EXISTS").
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}))
			},
			expectError: false,
			expectedLen: 0,
		},
		{
			name: "database query error",
			mockSetup: func() {
				mock.ExpectQuery("^SELECT (.+) FROM repositories r WHERE EXISTS").
					WillReturnError(fmt.Errorf("db connection lost"))
			},
			expectError: true,
			checkErrMsg: "query:",
		},
		{
			name: "row scan error - wrong id type",
			mockSetup: func() {
				mock.ExpectQuery("^SELECT (.+) FROM repositories r WHERE EXISTS").
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
				mock.ExpectQuery("^SELECT (.+) FROM repositories r WHERE EXISTS").
					WillReturnRows(rows)
			},
			expectError: true,
			checkErrMsg: "scan:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			result, err := repo.GetAll(context.Background())

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
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

func TestRepositoryRepository_GetByName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &RepositoryRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	now := time.Now()

	tests := []struct {
		name        string
		repoName    string
		mockSetup   func(name string)
		expectError bool
		checkErrMsg string
	}{
		{
			name:     "success",
			repoName: "golang/go",
			mockSetup: func(name string) {
				mock.ExpectQuery("^SELECT id, full_name, last_seen_tag, updated_at FROM repositories").
					WithArgs(name).
					WillReturnRows(pgxmock.NewRows([]string{"id", "full_name", "last_seen_tag", "updated_at"}).
						AddRow(int64(1), name, "v1.25.0", now))
			},
			expectError: false,
		},
		{
			name:     "not found error",
			repoName: "unknown/repo",
			mockSetup: func(name string) {
				mock.ExpectQuery("^SELECT id, full_name, last_seen_tag, updated_at FROM repositories").
					WithArgs(name).
					WillReturnError(pgx.ErrNoRows)
			},
			expectError: true,
			checkErrMsg: model.ErrRepositoryNotFound.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.repoName)
			result, err := repo.GetByName(context.Background(), tt.repoName)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.repoName, result.FullName)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepositoryRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &RepositoryRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	now := time.Now()

	tests := []struct {
		name        string
		inputRepo   *model.Repository
		mockSetup   func(r *model.Repository)
		expectError bool
		checkErrMsg string
	}{
		{
			name: "success create",
			inputRepo: &model.Repository{
				FullName:    "golang/go",
				LastSeenTag: "v1.25.0",
			},
			mockSetup: func(r *model.Repository) {
				mock.ExpectQuery("^INSERT INTO repositories").
					WithArgs(r.FullName, r.LastSeenTag).
					WillReturnRows(pgxmock.NewRows([]string{"id", "updated_at"}).
						AddRow(int64(1), now))
			},
			expectError: false,
		},
		{
			name: "database error on insert",
			inputRepo: &model.Repository{
				FullName:    "golang/go",
				LastSeenTag: "v1.25.0",
			},
			mockSetup: func(r *model.Repository) {
				mock.ExpectQuery("^INSERT INTO repositories").
					WithArgs(r.FullName, r.LastSeenTag).
					WillReturnError(fmt.Errorf("unique constraint violation"))
			},
			expectError: true,
			checkErrMsg: "insert:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.inputRepo)
			err := repo.Create(context.Background(), tt.inputRepo)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, tt.inputRepo.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepositoryRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := &RepositoryRepository{
		db:     mock,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	tests := []struct {
		name        string
		inputRepo   *model.Repository
		mockSetup   func(r *model.Repository)
		expectError bool
		checkErrMsg string
	}{
		{
			name: "success update",
			inputRepo: &model.Repository{
				ID:          1,
				LastSeenTag: "v1.26.0",
			},
			mockSetup: func(r *model.Repository) {
				mock.ExpectExec("^UPDATE repositories").
					WithArgs(r.LastSeenTag, r.ID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			expectError: false,
		},
		{
			name: "not found - zero rows affected",
			inputRepo: &model.Repository{
				ID:          999,
				LastSeenTag: "v1.26.0",
			},
			mockSetup: func(r *model.Repository) {
				mock.ExpectExec("^UPDATE repositories").
					WithArgs(r.LastSeenTag, r.ID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			expectError: true,
			checkErrMsg: model.ErrRepositoryNotFound.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.inputRepo)
			err := repo.Update(context.Background(), tt.inputRepo)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.checkErrMsg)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
