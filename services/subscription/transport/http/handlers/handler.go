package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	usecase2 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/domain/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-ReilEgor/services/subscription/transport/http/middleware"
)

type Handler struct {
	userUC usecase2.UserUseCase
	repoUC usecase2.RepositoryUseCase
	logger *slog.Logger
	apiKey string
}

func NewHandler(userUC usecase2.UserUseCase, repoUC usecase2.RepositoryUseCase, apiKey string) *Handler {
	return &Handler{
		userUC: userUC,
		repoUC: repoUC,
		logger: slog.With(slog.String("component", "handler")),
		apiKey: apiKey,
	}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.StaticFile("/", "./static/index.html")

	api := router.Group("/api/v1")
	{
		api.GET("/confirm/:token", h.Confirm)
		api.GET("/unsubscribe/:token", h.UnsubscribeByToken)

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(h.apiKey))
		{
			protected.POST("/subscribe", h.Subscribe)
			protected.GET("/subscriptions", h.ListSubscriptions)
		}
	}

	internal := router.Group("/internal")
	internal.Use(middleware.AuthMiddleware(h.apiKey))
	{
		internal.POST("/repositories/tag", h.UpdateTag)
	}
}
