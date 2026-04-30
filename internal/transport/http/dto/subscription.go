package dto

import "time"

// SubscriptionResponse represents a user's subscription record.
type SubscriptionResponse struct {
	ID             int64     `json:"id" example:"1"`                            // Unique identifier of the subscription.
	Email          string    `json:"email" example:"user@example.com"`          // Subscriber's email address.
	RepositoryName string    `json:"repository_name" example:"golang/go"`       // The name of the repository (extracted from FullName).
	Confirmed      bool      `json:"confirmed"`                                 // Indicates whether the subscription has been confirmed by the user.
	LastSeenTag    string    `json:"last_seen_tag"`                             // The latest tag seen for this subscription, useful for tracking updates.
	CreatedAt      time.Time `json:"created_at" example:"2026-04-08T18:00:00Z"` // Timestamp when the subscription was created.
}

// CreateSubscriptionRequest defines the input payload for creating a new subscription.
type CreateSubscriptionRequest struct {
	Email      string `json:"email" binding:"required,email" example:"user@example.com"` // User email address for notifications.
	Repository string `json:"repository" binding:"required" example:"golang/go"`         // GitHub repository full name (owner/repo).
}

type CreateSubscriptionResponse struct {
	Message string `json:"message" example:"Confirmation email sent"` // Status message indicating the result of the subscription creation.
}

// DeleteSubscriptionRequest defines the params for removing an existing subscription.
type DeleteSubscriptionRequest struct {
	Email      string `json:"email" binding:"required,email" example:"user@example.com"` // Email address associated with the subscription.
	Repository string `json:"repository" binding:"required" example:"golang/go"`         // Repository name to unsubscribe from.
}

// DeleteSubscriptionResponse defines the output after a successful unsubscription.
type DeleteSubscriptionResponse struct {
	Message string `json:"message" example:"Subscription deleted successfully"` // Status message confirming the operation.
}

// ListSubscriptionsResponse represents a collection of subscriptions.
type ListSubscriptionsResponse struct {
	Subscriptions []SubscriptionResponse `json:"subscriptions"`     // List of the user's subscriptions.
	Total         int                    `json:"total" example:"2"` // Good practice to return count.
}
