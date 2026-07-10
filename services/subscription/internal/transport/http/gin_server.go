package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/shared/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/domain/usecase"
	handler "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/http/handlers"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/internal/transport/http/middleware"
)

type GinServer struct {
	router          *gin.Engine
	userUC          usecase.UserUseCase
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func NewGinServer(
	userUC usecase.UserUseCase,
	repoUC usecase.RepositoryUseCase,
	redisClient *redis.Client,
	httpCfg config.HTTPConfig,
	appCfg config.AppConfig,
) *GinServer {
	router := gin.New()
	logger := slog.With(slog.String("component", "gin_server"))
	middleware.SetupMiddleware(router, logger, redisClient, appCfg.RateLimit, httpCfg.RequestTimeout)

	s := &GinServer{
		router:          router,
		userUC:          userUC,
		logger:          logger,
		shutdownTimeout: httpCfg.ShutdownTimeout,
	}

	h := handler.NewHandler(userUC, repoUC, appCfg.APIKey)
	h.InitRoutes(s.router)

	return s
}

func (s *GinServer) Run(ctx context.Context, port string) error {
	srv := &http.Server{Addr: port, Handler: s.router}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("forced shutdown", "error", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server run: %w", err)
	}
	return nil
}
