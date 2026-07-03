package contracts

// Queue names for subscription lifecycle events.
const (
	QueueSubscriptionActivated   = "subscription.activated"
	QueueUnsubscriptionActivated = "unsubscription.activated"
)

// SubscriptionActivatedEvent is published by subscription when a user confirms their subscription.
// Consumed by tracking to start monitoring the repository for that subscriber.
type SubscriptionActivatedEvent struct {
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	RequestID string `json:"request_id,omitempty"`
}

// UnsubscriptionActivatedEvent is published by subscription when a user unsubscribes.
// Consumed by tracking to stop monitoring the repository for that subscriber.
type UnsubscriptionActivatedEvent struct {
	Email     string `json:"email"`
	RepoName  string `json:"repo_name"`
	RequestID string `json:"request_id,omitempty"`
}

// TagUpdatedEvent is published by tracking when a new release tag is detected.
// Consumed by subscription to update the stored tag for a repository.
type TagUpdatedEvent struct {
	FullName string `json:"full_name"`
	Tag      string `json:"tag"`
}
