package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ReilEgor/RepoNotifier/internal/domain/model"
	"github.com/ReilEgor/RepoNotifier/internal/domain/service"
	"github.com/ReilEgor/RepoNotifier/internal/transport/http/dto"
)

const (
	timeoutSubscribe   = 10 * time.Second
	timeoutUnsubscribe = 3 * time.Second
	timeoutConfirm     = 3 * time.Second
	timeoutList        = 3 * time.Second
)

const (
	errInvalidRequestBody  = "invalid request body"
	errFailedToSubscribe   = "failed to subscribe"
	errFailedToUnsubscribe = "failed to unsubscribe"
	errFailedToList        = "failed to list subscriptions"
)

var (
	ErrInvalidEmailFormat   = errors.New("invalid email format")
	ErrInvalidRepoFormat    = errors.New("invalid repository format (expected 'owner/repo')")
	ErrEmailRequired        = errors.New("email is required")
	ErrAlreadySubscribed    = errors.New("user is already subscribed to this repository")
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

var repoRegex = regexp.MustCompile(`^[a-zA-Z0-9-._]{1,100}/[a-zA-Z0-9-._]{1,100}$`)

func validateEmail(email string) error {
	if _, err := mail.ParseAddress(strings.TrimSpace(email)); err != nil {
		return ErrInvalidEmailFormat
	}
	return nil
}

func validateSubscription(email, repo string) []string {
	var errs []string
	if err := validateEmail(email); err != nil {
		errs = append(errs, err.Error())
	}
	if !repoRegex.MatchString(strings.TrimSpace(repo)) {
		errs = append(errs, ErrInvalidRepoFormat.Error())
	}
	return errs
}

func (h *Handler) handleTokenAction(
	c *gin.Context,
	timeout time.Duration,
	action func(ctx context.Context, token string) error,
	notFoundMsg string,
	internalMsg string,
) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	if err := action(ctx, token); err != nil {
		if errors.Is(err, model.ErrInvalidToken) {
			c.JSON(http.StatusNotFound, gin.H{"error": notFoundMsg})
			return
		}
		h.logger.ErrorContext(ctx, internalMsg, slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": internalMsg})
		return
	}
}

// Subscribe godoc
//
//	@Summary		Subscribe to a repository
//	@Description	Create a pending subscription and send a confirmation email.
//	@Tags			subscriptions
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateSubscriptionRequest	true	"Subscription details"
//	@Success		202		{object}	dto.CreateSubscriptionResponse
//	@Failure		400		{object}	map[string]string	"Invalid request body or validation errors"
//	@Failure		404		{object}	map[string]string	"Repository not found"
//	@Failure		409		{object}	map[string]string	"Already subscribed"
//	@Failure		503		{object}	map[string]string	"GitHub API unavailable"
//	@Router			/subscribe [post].
func (h *Handler) Subscribe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutSubscribe)
	defer cancel()

	log := h.logger.With(slog.String("handler", "Subscribe"))

	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WarnContext(ctx, errInvalidRequestBody, slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", errInvalidRequestBody, err.Error())})
		return
	}

	if errs := validateSubscription(req.Email, req.Repository); len(errs) > 0 {
		log.WarnContext(ctx, "validation failed",
			slog.String("email", req.Email),
			slog.String("repo", req.Repository),
			slog.Any("errors", errs),
		)
		c.JSON(http.StatusBadRequest, gin.H{"errors": errs})
		return
	}

	if err := h.subscriptionUC.Subscribe(ctx, req.Email, req.Repository); err != nil {
		switch {
		case errors.Is(err, service.ErrRepositoryNotFound):
			log.WarnContext(ctx, "repository not found",
				slog.String("email", req.Email),
				slog.String("repo", req.Repository),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrAlreadySubscribed):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrGitHubUnavailable), errors.Is(err, service.ErrRateLimitExceeded):
			c.JSON(
				http.StatusServiceUnavailable,
				gin.H{"error": "GitHub API is currently unavailable, please try again later"},
			)
		default:
			log.ErrorContext(ctx, errFailedToSubscribe,
				slog.String("email", req.Email),
				slog.String("repo", req.Repository),
				slog.String("error", err.Error()),
			)
			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("%s: %s", errFailedToSubscribe, err.Error())},
			)
		}
		return
	}

	log.InfoContext(ctx, "subscribed successfully",
		slog.String("email", req.Email),
		slog.String("repo", req.Repository),
	)
	c.JSON(http.StatusAccepted, dto.CreateSubscriptionResponse{
		Message: "Subscription initiated. Please check your email to confirm.",
	})
}

// UnsubscribeByToken godoc
//
//	@Summary		Unsubscribe via token
//	@Description	Remove a subscription using the one-time token from the unsubscribe link.
//	@Tags			subscriptions
//	@Produce		json
//	@Param			token	path		string	true	"Unsubscribe token"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string	"Token is required"
//	@Failure		404		{object}	map[string]string	"Invalid or expired token"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/unsubscribe/{token} [get].
func (h *Handler) UnsubscribeByToken(c *gin.Context) {
	h.handleTokenAction(
		c,
		timeoutUnsubscribe,
		h.subscriptionUC.UnsubscribeByToken,
		"invalid or expired unsubscribe link",
		errFailedToUnsubscribe,
	)
	if !c.Writer.Written() {
		c.JSON(http.StatusOK, gin.H{"message": "You have been successfully unsubscribed"})
	}
}

// ListSubscriptions godoc
//
//	@Summary		Get all subscriptions by email
//	@Description	Retrieve a list of all subscriptions (confirmed and pending) for a given email.
//	@Tags			subscriptions
//	@Produce		json
//	@Param			email	query		string	true	"User email address"
//	@Success		200		{object}	dto.ListSubscriptionsResponse
//	@Failure		400		{object}	map[string]string	"Email is required or invalid"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/subscriptions [get].
func (h *Handler) ListSubscriptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutList)
	defer cancel()

	log := h.logger.With(
		slog.String("op", "Handler.ListSubscriptions"),
		slog.String("handler", "ListSubscriptions"),
	)

	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		log.WarnContext(ctx, "email query param missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrEmailRequired.Error()})
		return
	}

	if err := validateEmail(email); err != nil {
		log.WarnContext(ctx, "invalid email format", slog.String("email", email))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subs, err := h.subscriptionUC.ListByEmail(ctx, email)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch subscriptions",
			slog.String("email", email),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errFailedToList})
		return
	}

	responseSubs := make([]dto.SubscriptionResponse, 0, len(subs))
	for _, s := range subs {
		responseSubs = append(responseSubs, dto.SubscriptionResponse{
			ID:             s.ID,
			Email:          email,
			RepositoryName: s.RepositoryName,
			CreatedAt:      s.CreatedAt,
			LastSeenTag:    s.LastSeenTag,
			Confirmed:      s.Confirmed,
		})
	}

	log.InfoContext(ctx, "successfully listed subscriptions",
		slog.String("email", email),
		slog.Int("count", len(responseSubs)),
	)

	c.JSON(http.StatusOK, dto.ListSubscriptionsResponse{
		Subscriptions: responseSubs,
		Total:         len(responseSubs),
	})
}

// Confirm godoc
//
//	@Summary		Confirm email subscription
//	@Description	Confirm a pending subscription using the token sent via email.
//	@Tags			subscriptions
//	@Produce		json
//	@Param			token	path		string				true	"Confirmation token"
//	@Success		200		{object}	map[string]string	"subscription confirmed successfully"
//	@Failure		400		{object}	map[string]string	"Token is required"
//	@Failure		404		{object}	map[string]string	"Invalid or expired token"
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/confirm/{token} [get].
func (h *Handler) Confirm(c *gin.Context) {
	h.handleTokenAction(
		c,
		timeoutConfirm,
		h.subscriptionUC.Confirm,
		"invalid or expired token",
		"failed to confirm subscription",
	)
	if !c.Writer.Written() {
		c.JSON(http.StatusOK, gin.H{"message": "subscription confirmed successfully"})
	}
}
