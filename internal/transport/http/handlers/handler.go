package handlers

import (
	"log/slog"
	"net/http"

	"github.com/ReilEgor/RepoNotifier/internal/domain/usecase"
	"github.com/ReilEgor/RepoNotifier/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handler struct {
	subscriptionUC usecase.SubscriptionUseCase
	logger         *slog.Logger
	apiKey         string
}

func NewHandler(subscriptionUC usecase.SubscriptionUseCase, apiKey string) *Handler {
	return &Handler{
		subscriptionUC: subscriptionUC,
		logger:         slog.With(slog.String("component", "handler")),
		apiKey:         apiKey,
	}
}

func (h *Handler) InitRoutes(router *gin.Engine) {
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
}
