package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	redisClient "github.com/redis/go-redis/v9"

	cacheRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/cache/redis"
	servicesRealizationGitHub "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/infrastructure/clients/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/mocks"
	repositoryRealization "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/repository/postgres"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/handlers"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/usecase"
)

const testAPIKey = "test-api-key"

const (
	testEmail = "test@example.com"
	testRepo  = "golang/go"
	testTag   = "v1.22.0"
)

type APITestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc

	dbPool         *pgxpool.Pool
	pgContainer    *postgres.PostgresContainer
	redisContainer *redis.RedisContainer
	redisClient    *redisClient.Client
	router         *gin.Engine

	mockGitHub *mocks.GitHubClient
	mockSMTP   *mocks.EmailService
}

func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (s *APITestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	pgContainer, err := postgres.Run(
		s.ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	s.Require().NoError(err, "failed to start PostgreSQL container")
	s.pgContainer = pgContainer

	connStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	s.Require().NoError(err)
	s.Require().NoError(runMigrations(connStr, "../../migrations"))

	pool, err := pgxpool.New(s.ctx, connStr)
	s.Require().NoError(err, "failed to create pgxpool")
	s.dbPool = pool

	redisContainer, err := redis.Run(
		s.ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(20*time.Second),
		),
	)
	s.Require().NoError(err, "failed to start Redis container")
	s.redisContainer = redisContainer

	redisURL, err := redisContainer.ConnectionString(s.ctx)
	s.Require().NoError(err)

	opt, err := redisClient.ParseURL(redisURL)
	s.Require().NoError(err)

	s.redisClient = redisClient.NewClient(opt)
	s.Require().NoError(s.redisClient.Ping(s.ctx).Err(), "redis ping failed")
}

func (s *APITestSuite) TearDownSuite() {
	if s.redisClient != nil {
		s.NoError(s.redisClient.Close())
	}
	if s.redisContainer != nil {
		s.NoError(s.redisContainer.Terminate(s.ctx))
	}
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.pgContainer != nil {
		s.NoError(s.pgContainer.Terminate(s.ctx))
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *APITestSuite) SetupTest() {
	s.truncateTables()
	s.Require().NoError(s.redisClient.FlushAll(s.ctx).Err())

	s.mockGitHub = new(mocks.GitHubClient)
	s.mockSMTP = new(mocks.EmailService)

	s.buildRouter()
}

func (s *APITestSuite) TearDownTest() {
	s.mockGitHub.AssertExpectations(s.T())
	s.mockSMTP.AssertExpectations(s.T())
}

func (s *APITestSuite) buildRouter() {
	cache := cacheRealization.NewCache(s.redisClient)
	cachedGitHub := servicesRealizationGitHub.NewCachedGitHubClient(s.mockGitHub, cache)

	repoRepo := repositoryRealization.NewRepositoryRepository(s.dbPool)
	userRepo := repositoryRealization.NewUserRepository(s.dbPool)
	subsRepo := repositoryRealization.NewSubscriptionRepository(s.dbPool)

	repoUseCase := usecase.NewRepositoryUseCase(repoRepo, cachedGitHub)
	userUseCase := usecase.NewUserUseCase(subsRepo, userRepo, repoUseCase, s.mockSMTP)

	handler := handlers.NewHandler(userUseCase, testAPIKey)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.InitRoutes(router)
	s.router = router
}

func (s *APITestSuite) truncateTables() {
	tables := []string{
		"subscriptions",
		"users",
		"repositories",
	}

	for _, t := range tables {
		query := fmt.Sprintf(
			`TRUNCATE TABLE "%s" RESTART IDENTITY CASCADE`,
			t,
		)

		_, err := s.dbPool.Exec(s.ctx, query)
		s.Require().NoError(err, "failed to truncate table "+t)
	}
}

func runMigrations(connStr, migrationsPath string) error {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()

		if sourceErr != nil {
			log.Printf("failed to close migration source: %v", sourceErr)
		}

		if dbErr != nil {
			log.Printf("failed to close migration db: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (s *APITestSuite) doRequest(method, path string, body io.Reader) *httptest.ResponseRecorder {
	return s.doRequestWithKey(method, path, body, testAPIKey)
}

func (s *APITestSuite) doRequestNoAuth(method, path string, body io.Reader) *httptest.ResponseRecorder {
	return s.doRequestWithKey(method, path, body, "")
}

func (s *APITestSuite) doRequestWithKey(method, path string, body io.Reader, key string) *httptest.ResponseRecorder {
	s.T().Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(s.ctx, method, path, body)
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	s.router.ServeHTTP(w, req)
	return w
}

func (s *APITestSuite) seedSubscription(email, repoName, tag string, confirmed bool) string {
	s.T().Helper()
	token := fmt.Sprintf("test-token-%d", time.Now().UnixNano())

	_, err := s.dbPool.Exec(s.ctx,
		`INSERT INTO users (email) VALUES ($1) ON CONFLICT (email) DO NOTHING`, email)
	s.Require().NoError(err)

	_, err = s.dbPool.Exec(s.ctx,
		`INSERT INTO repositories (full_name, last_seen_tag) VALUES ($1, $2)
        ON CONFLICT (full_name) DO NOTHING`, repoName, tag)
	s.Require().NoError(err)

	_, err = s.dbPool.Exec(s.ctx, `
       INSERT INTO subscriptions (user_id, repository_id, token, is_confirmed)
       SELECT u.id, r.id, $3, $4
       FROM users u, repositories r
       WHERE u.email = $1 AND r.full_name = $2`,
		email, repoName, token, confirmed)
	s.Require().NoError(err)

	return token
}
