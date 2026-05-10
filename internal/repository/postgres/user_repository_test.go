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

func TestUserRepository_GetByEmail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &UserRepository{
		db:     mock,
		logger: discardLogger,
	}

	userEmail := "test@example.com"
	now := time.Now()

	tests := []struct {
		name        string
		email       string
		mockSetup   func(email string)
		expectError bool
		errorIs     error
		checkErrMsg string
		expected    model.User
	}{
		{
			name:  "success",
			email: userEmail,
			mockSetup: func(email string) {
				mock.ExpectQuery("^SELECT id, email, created_at FROM users WHERE email = \\$1").
					WithArgs(email).
					WillReturnRows(pgxmock.NewRows([]string{"id", "email", "created_at"}).
						AddRow(int64(1), email, now))
			},
			expectError: false,
			expected: model.User{
				ID:        1,
				Email:     userEmail,
				CreatedAt: now,
			},
		},
		{
			name:  "user not found",
			email: "unknown@example.com",
			mockSetup: func(email string) {
				mock.ExpectQuery("^SELECT (.+) FROM users WHERE email = \\$1").
					WithArgs(email).
					WillReturnError(pgx.ErrNoRows)
			},
			expectError: true,
			errorIs:     model.ErrUserNotFound,
		},
		{
			name:  "database error",
			email: userEmail,
			mockSetup: func(email string) {
				mock.ExpectQuery("^SELECT (.+) FROM users").
					WithArgs(email).
					WillReturnError(fmt.Errorf("internal db error"))
			},
			expectError: true,
			checkErrMsg: "query row:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.email)

			result, err := repo.GetByEmail(context.Background(), tt.email)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorIs != nil {
					require.ErrorIs(t, err, tt.errorIs)
				}
				if tt.checkErrMsg != "" {
					assert.Contains(t, err.Error(), tt.checkErrMsg)
				}
				assert.Equal(t, model.User{}, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.Email, result.Email)
				assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &UserRepository{
		db:     mock,
		logger: discardLogger,
	}

	now := time.Now()

	tests := []struct {
		name        string
		inputUser   *model.User
		mockSetup   func(u *model.User)
		expectError bool
		checkErrMsg string
	}{
		{
			name: "success create new user",
			inputUser: &model.User{
				Email: "test@example.com",
			},
			mockSetup: func(u *model.User) {
				mock.ExpectQuery("^INSERT INTO users").
					WithArgs(u.Email).
					WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).
						AddRow(int64(1), now))
			},
			expectError: false,
		},
		{
			name: "database error on insert (e.g. duplicate email)",
			inputUser: &model.User{
				Email: "duplicate@example.com",
			},
			mockSetup: func(u *model.User) {
				mock.ExpectQuery("^INSERT INTO users").
					WithArgs(u.Email).
					WillReturnError(fmt.Errorf("unique constraint violation"))
			},
			expectError: true,
			checkErrMsg: "insert:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(tt.inputUser)

			err := repo.Create(context.Background(), tt.inputUser)

			if tt.expectError {
				require.Error(t, err)
				if tt.checkErrMsg != "" {
					assert.Contains(t, err.Error(), tt.checkErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(1), tt.inputUser.ID)
				assert.Equal(t, now, tt.inputUser.CreatedAt)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
