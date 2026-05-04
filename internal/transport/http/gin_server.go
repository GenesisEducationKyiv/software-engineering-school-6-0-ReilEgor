package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/domain/usecase"
	handler "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/handlers"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/internal/transport/http/middleware"
)

type GinServer struct {
	router         *gin.Engine
	subscriptionUC usecase.SubscriptionUseCase
	logger         *slog.Logger
}

func NewGinServer(
	subscriptionUC usecase.SubscriptionUseCase,
	redisClient *redis.Client,
	apiKey config.APIKeyType,
) *GinServer {
	router := gin.New()
	logger := slog.With(slog.String("component", "gin_server"))
	middleware.SetupMiddleware(router, logger, redisClient)

	s := &GinServer{
		router:         router,
		subscriptionUC: subscriptionUC,
		logger:         logger,
	}

	h := handler.NewHandler(subscriptionUC, string(apiKey))
	h.InitRoutes(s.router)

	return s
}

func (s *GinServer) Run(ctx context.Context, port string) error {
	srv := &http.Server{Addr: port, Handler: s.router}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
